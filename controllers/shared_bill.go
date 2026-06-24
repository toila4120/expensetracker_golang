package controllers

import (
	"expensetracker/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CreateBillInput struct {
	PayerID         uint              `json:"payer_id" binding:"required"`
	Amount          int               `json:"amount" binding:"required,gt=0"`
	Category        string            `json:"category" binding:"required,oneof=food transport food_transport shopping entertainment education health other"`
	Description     string            `json:"description"`
	SplitMethod     string            `json:"split_method" binding:"required,oneof=equal percentage custom"`
	TransactionDate time.Time         `json:"transaction_date"`
	Splits          []BillSplitInput  `json:"splits" binding:"required,min=1"`
}

type QuickBillInput struct {
	PayerID         uint              `json:"payer_id" binding:"required"`
	Amount          int               `json:"amount" binding:"required,gt=0"`
	Category        string            `json:"category" binding:"required,oneof=food transport food_transport shopping entertainment education health other"`
	Description     string            `json:"description"`
	SplitMethod     string            `json:"split_method" binding:"required,oneof=equal percentage custom"`
	TransactionDate time.Time         `json:"transaction_date"`
	Members         []QuickMemberInput `json:"members" binding:"required,min=1"`
}

type QuickMemberInput struct {
	UserID    *uint  `json:"user_id"`
	GuestName string `json:"guest_name"`
	Amount    int    `json:"amount" binding:"required,gt=0"`
}

type BillSplitInput struct {
	GroupMemberID uint `json:"group_member_id" binding:"required"`
	Amount        int  `json:"amount" binding:"required,gt=0"`
}

func CreateSharedBill(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("currentUserID").(uint)
		groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID nhóm không hợp lệ"})
			return
		}

		var input CreateBillInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ", "details": err.Error()})
			return
		}

		var group models.Group
		if err := db.First(&group, groupID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy nhóm"})
			return
		}

		var isMember int64
		db.Model(&models.GroupMember{}).Where("group_id = ? AND user_id = ?", group.ID, userID).Count(&isMember)
		if isMember == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "Bạn không phải thành viên của nhóm này"})
			return
		}

		var payerMember models.GroupMember
		if err := db.Where("id = ? AND group_id = ?", input.PayerID, group.ID).First(&payerMember).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Người trả tiền không phải thành viên của nhóm"})
			return
		}

		memberIDs := make(map[uint]bool)
		var members []models.GroupMember
		db.Where("group_id = ?", group.ID).Find(&members)
		for _, m := range members {
			memberIDs[m.ID] = true
		}

		for _, split := range input.Splits {
			if !memberIDs[split.GroupMemberID] {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Thành viên chia tiền không phải thành viên của nhóm"})
				return
			}
		}

		seenMembers := make(map[uint]bool)
		totalSplit := 0
		for _, split := range input.Splits {
			if seenMembers[split.GroupMemberID] {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Một thành viên chỉ được xuất hiện một lần trong danh sách chia tiền"})
				return
			}
			seenMembers[split.GroupMemberID] = true
			totalSplit += split.Amount
		}

		if totalSplit != input.Amount {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":        "Tổng tiền chia không khớp với tổng hóa đơn",
				"bill_amount":  input.Amount,
				"split_total":  totalSplit,
			})
			return
		}

		txDate := input.TransactionDate
		if txDate.IsZero() {
			txDate = time.Now()
		}

		tx := db.Begin()

		sharedBill := models.SharedBill{
			GroupID:         group.ID,
			PayerID:         payerMember.ID,
			CreatorID:       userID,
			Amount:          input.Amount,
			Category:        input.Category,
			Description:     input.Description,
			SplitMethod:     input.SplitMethod,
			TransactionDate: txDate,
		}
		if err := tx.Create(&sharedBill).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo hóa đơn"})
			return
		}

		for _, split := range input.Splits {
			billSplit := models.BillSplit{
				SharedBillID:  sharedBill.ID,
				GroupMemberID: split.GroupMemberID,
				Amount:        split.Amount,
				IsSettled:     false,
			}
			if err := tx.Create(&billSplit).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo chi tiết chia tiền"})
				return
			}
		}

		tx.Commit()

		c.JSON(http.StatusCreated, gin.H{
			"message": "Tạo hóa đơn thành công",
			"data":    sharedBill,
		})
	}
}

func GetSharedBills(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("currentUserID").(uint)
		groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID nhóm không hợp lệ"})
			return
		}

		var isMember int64
		db.Model(&models.GroupMember{}).Where("group_id = ? AND user_id = ?", groupID, userID).Count(&isMember)
		if isMember == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "Bạn không phải thành viên của nhóm này"})
			return
		}

		var bills []models.SharedBill
		if err := db.Where("group_id = ?", groupID).Order("transaction_date DESC").Find(&bills).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy danh sách hóa đơn"})
			return
		}

		type BillResponse struct {
			models.SharedBill
			PayerName    string            `json:"payer_name"`
			CreatorName  string            `json:"creator_name"`
			Splits       []models.BillSplit `json:"splits"`
		}

		var responses []BillResponse
		for _, bill := range bills {
			var payerMember models.GroupMember
			db.First(&payerMember, bill.PayerID)

			var creator models.User
			db.First(&creator, bill.CreatorID)

			var splits []models.BillSplit
			db.Where("shared_bill_id = ?", bill.ID).Find(&splits)

			payerName := ""
			if payerMember.UserID != nil {
				var user models.User
				db.First(&user, *payerMember.UserID)
				payerName = user.Username
			} else {
				payerName = payerMember.GuestName
			}

			responses = append(responses, BillResponse{
				SharedBill:  bill,
				PayerName:   payerName,
				CreatorName: creator.Username,
				Splits:      splits,
			})
		}

		c.JSON(http.StatusOK, gin.H{"data": responses})
	}
}

func CreateQuickBill(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("currentUserID").(uint)

		var input QuickBillInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ", "details": err.Error()})
			return
		}

		// Validate: at least 1 member in splits
		if len(input.Members) < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cần ít nhất 1 thành viên trong danh sách chia tiền"})
			return
		}

		// Validate total split matches bill amount
		totalSplit := 0
		for _, m := range input.Members {
			if m.Amount <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Số tiền chia phải lớn hơn 0"})
				return
			}
			totalSplit += m.Amount
		}
		if totalSplit != input.Amount {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":       "Tổng tiền chia không khớp với tổng hóa đơn",
				"bill_amount": input.Amount,
				"split_total": totalSplit,
			})
			return
		}

		// Validate each member has user_id or guest_name
		for _, m := range input.Members {
			if m.UserID == nil && m.GuestName == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Mỗi thành viên phải có user_id hoặc guest_name"})
				return
			}
		}

		// Collect all unique member identifiers (including payer)
		type memberKey struct {
			UserID    *uint
			GuestName string
		}
		memberMap := make(map[memberKey]bool)
		
		// Add payer (payer_id is user.id)
		payerUserID := input.PayerID
		memberMap[memberKey{UserID: &payerUserID}] = true

		// Add all split members
		for _, m := range input.Members {
			memberMap[memberKey{UserID: m.UserID, GuestName: m.GuestName}] = true
		}

		tx := db.Begin()

		// Step 1: Find or create peer_to_peer group with exactly these members
		var existingGroup models.Group
		
		// Build list of all member user_ids and guest_names
		var allUserIDs []uint
		var allGuestNames []string
		for key := range memberMap {
			if key.UserID != nil {
				allUserIDs = append(allUserIDs, *key.UserID)
			} else {
				allGuestNames = append(allGuestNames, key.GuestName)
			}
		}

		// Try to find existing peer_to_peer group with exact same members
		foundGroup := false
		
		if len(allUserIDs) > 0 {
			var candidateGroups []models.Group
			// Find all peer_to_peer groups that have members with these user_ids
			err := db.Raw(`
				SELECT DISTINCT g.* FROM groups g
				INNER JOIN group_members gm ON gm.group_id = g.id
				WHERE g.type = 'peer_to_peer' 
				AND g.deleted_at IS NULL
				AND gm.user_id IN (?)
				GROUP BY g.id
				HAVING (
					SELECT COUNT(*) FROM group_members 
					WHERE group_id = g.id AND deleted_at IS NULL
				) = ?
			`, allUserIDs, len(memberMap)).Scan(&candidateGroups).Error
			
			if err == nil {
				for _, candidate := range candidateGroups {
					// Verify exact match
					var candidateMembers []models.GroupMember
					db.Where("group_id = ?", candidate.ID).Find(&candidateMembers)
					
					if len(candidateMembers) != len(memberMap) {
						continue
					}
					
					match := true
					candidateUserIDs := make(map[uint]bool)
					candidateGuestNames := make(map[string]bool)
					
					for _, cm := range candidateMembers {
						if cm.UserID != nil {
							candidateUserIDs[*cm.UserID] = true
						} else {
							candidateGuestNames[cm.GuestName] = true
						}
					}
					
					for key := range memberMap {
						if key.UserID != nil {
							if !candidateUserIDs[*key.UserID] {
								match = false
								break
							}
						} else {
							if !candidateGuestNames[key.GuestName] {
								match = false
								break
							}
						}
					}
					
					if match {
						existingGroup = candidate
						foundGroup = true
						break
					}
				}
			}
		}

		var groupID uint

		if foundGroup {
			groupID = existingGroup.ID
		} else {
			// Create new peer_to_peer group
			// Generate name from member names
			var nameParts []string
			for key := range memberMap {
				if key.UserID != nil {
					var user models.User
					if db.First(&user, *key.UserID).Error == nil {
						nameParts = append(nameParts, user.Username)
					}
				} else {
					nameParts = append(nameParts, key.GuestName)
				}
			}
			// Join first 3 names
			groupName := ""
			for i, name := range nameParts {
				if i > 0 {
					groupName += ", "
				}
				if i >= 3 {
					groupName += "..."
					break
				}
				groupName += name
			}

			newGroup := models.Group{
				Name:        groupName,
				Description: "Nhóm thanh toán nhanh",
				Type:        "peer_to_peer",
				CreatedBy:   userID,
			}
			if err := tx.Create(&newGroup).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo nhóm"})
				return
			}
			groupID = newGroup.ID

			// Create group members
			for key := range memberMap {
				member := models.GroupMember{
					GroupID:   groupID,
					UserID:    key.UserID,
					GuestName: key.GuestName,
					Role:      "member",
				}
				if err := tx.Create(&member).Error; err != nil {
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể thêm thành viên vào nhóm"})
					return
				}
			}
		}

		// Step 2: Get all group members to map input members to group_member_ids
		var allMembers []models.GroupMember
		db.Where("group_id = ?", groupID).Find(&allMembers)

		memberIDMap := make(map[string]uint) // key: "user_id:X" or "guest_name:X"
		for _, m := range allMembers {
			if m.UserID != nil {
				memberIDMap["user_id:"+strconv.FormatUint(uint64(*m.UserID), 10)] = m.ID
			} else {
				memberIDMap["guest_name:"+m.GuestName] = m.ID
			}
		}

		// Step 3: Find payer's group_member_id
		payerGroupMemberID, exists := memberIDMap["user_id:"+strconv.FormatUint(uint64(payerUserID), 10)]
		if !exists {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy người trả tiền trong nhóm"})
			return
		}

		// Step 4: Create shared bill
		txDate := input.TransactionDate
		if txDate.IsZero() {
			txDate = time.Now()
		}

		sharedBill := models.SharedBill{
			GroupID:         groupID,
			PayerID:         payerGroupMemberID,
			CreatorID:       userID,
			Amount:          input.Amount,
			Category:        input.Category,
			Description:     input.Description,
			SplitMethod:     input.SplitMethod,
			TransactionDate: txDate,
		}
		if err := tx.Create(&sharedBill).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo hóa đơn"})
			return
		}

		// Step 5: Create bill splits
		for _, m := range input.Members {
			var memberKey string
			if m.UserID != nil {
				memberKey = "user_id:" + strconv.FormatUint(uint64(*m.UserID), 10)
			} else {
				memberKey = "guest_name:" + m.GuestName
			}

			groupMemberID, exists := memberIDMap[memberKey]
			if !exists {
				tx.Rollback()
				c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy thành viên trong nhóm"})
				return
			}

			billSplit := models.BillSplit{
				SharedBillID:  sharedBill.ID,
				GroupMemberID: groupMemberID,
				Amount:        m.Amount,
				IsSettled:     false,
			}
			if err := tx.Create(&billSplit).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo chi tiết chia tiền"})
				return
			}
		}

		tx.Commit()

		c.JSON(http.StatusCreated, gin.H{
			"message":  "Tạo hóa đơn thành công",
			"data":     sharedBill,
			"group_id": groupID,
		})
	}
}
