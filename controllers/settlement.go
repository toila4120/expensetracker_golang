package controllers

import (
	"expensetracker/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SettleInput struct {
	FromMemberID uint `json:"from_member_id" binding:"required"`
	ToMemberID   uint `json:"to_member_id" binding:"required"`
	Amount       int  `json:"amount" binding:"required,gt=0"`
}

func GetBalances(db *gorm.DB) gin.HandlerFunc {
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

		var members []models.GroupMember
		db.Where("group_id = ?", groupID).Find(&members)

		balances := make(map[uint]int)
		for _, m := range members {
			balances[m.ID] = 0
		}

		var bills []models.SharedBill
		db.Where("group_id = ?", groupID).Find(&bills)

		for _, bill := range bills {
			balances[bill.PayerID] += bill.Amount

			var splits []models.BillSplit
			db.Where("shared_bill_id = ? AND is_settled = ?", bill.ID, false).Find(&splits)
			for _, split := range splits {
				balances[split.GroupMemberID] -= split.Amount
			}
		}

		var settlements []models.Settlement
		db.Where("group_id = ?", groupID).Find(&settlements)
		for _, s := range settlements {
			balances[s.FromID] += s.Amount
			balances[s.ToID] -= s.Amount
		}

		type BalanceEntry struct {
			MemberID   uint   `json:"member_id"`
			UserID     *uint  `json:"user_id"`
			GuestName  string `json:"guest_name"`
			Username   string `json:"username"`
			Amount     int    `json:"amount"`
		}

		type OwesEntry struct {
			ToMemberID uint   `json:"to_member_id"`
			ToUsername  string `json:"to_username"`
			Amount     int    `json:"amount"`
		}

		type MemberBalance struct {
			MemberID  uint        `json:"member_id"`
			UserID    *uint       `json:"user_id"`
			GuestName string      `json:"guest_name"`
			Username  string      `json:"username"`
			Owes      []OwesEntry `json:"owes"`
			GetsBack  []OwesEntry `json:"gets_back"`
		}

		creditors := []struct {
			MemberID uint
			Amount   int
		}{}
		debtors := []struct {
			MemberID uint
			Amount   int
		}{}

		for memberID, amount := range balances {
			if amount > 0 {
				creditors = append(creditors, struct {
					MemberID uint
					Amount   int
				}{memberID, amount})
			} else if amount < 0 {
				debtors = append(debtors, struct {
					MemberID uint
					Amount   int
				}{memberID, -amount})
			}
		}

		settlementPlan := []struct {
			From   uint
			To     uint
			Amount int
		}{}

		i, j := 0, 0
		for i < len(debtors) && j < len(creditors) {
			d := debtors[i]
			cr := creditors[j]
			transfer := d.Amount
			if cr.Amount < transfer {
				transfer = cr.Amount
			}
			settlementPlan = append(settlementPlan, struct {
				From   uint
				To     uint
				Amount int
			}{d.MemberID, cr.MemberID, transfer})
			debtors[i].Amount -= transfer
			creditors[j].Amount -= transfer
			if debtors[i].Amount == 0 {
				i++
			}
			if creditors[j].Amount == 0 {
				j++
			}
		}

		memberMap := make(map[uint]models.GroupMember)
		for _, m := range members {
			memberMap[m.ID] = m
		}

		getUsername := func(m models.GroupMember) string {
			if m.UserID != nil {
				var user models.User
				db.First(&user, *m.UserID)
				return user.Username
			}
			return m.GuestName
		}

		resultMap := make(map[uint]*MemberBalance)
		for _, m := range members {
			resultMap[m.ID] = &MemberBalance{
				MemberID:  m.ID,
				UserID:    m.UserID,
				GuestName: m.GuestName,
				Username:  getUsername(m),
				Owes:      []OwesEntry{},
				GetsBack:  []OwesEntry{},
			}
		}

		for _, sp := range settlementPlan {
			fromMember := memberMap[sp.From]
			toMember := memberMap[sp.To]
			resultMap[sp.From].Owes = append(resultMap[sp.From].Owes, OwesEntry{
				ToMemberID: sp.To,
				ToUsername:  getUsername(toMember),
				Amount:     sp.Amount,
			})
			resultMap[sp.To].GetsBack = append(resultMap[sp.To].GetsBack, OwesEntry{
				ToMemberID: sp.From,
				ToUsername:  getUsername(fromMember),
				Amount:     sp.Amount,
			})
		}

		var result []MemberBalance
		for _, m := range members {
			result = append(result, *resultMap[m.ID])
		}

		c.JSON(http.StatusOK, gin.H{
			"group_id":   groupID,
			"group_name": func() string { var g models.Group; db.First(&g, groupID); return g.Name }(),
			"balances":   result,
		})
	}
}

func SettleDebt(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("currentUserID").(uint)
		groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID nhóm không hợp lệ"})
			return
		}

		var input SettleInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ", "details": err.Error()})
			return
		}

		var isMember int64
		db.Model(&models.GroupMember{}).Where("group_id = ? AND user_id = ?", groupID, userID).Count(&isMember)
		if isMember == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "Bạn không phải thành viên của nhóm này"})
			return
		}

		var fromMember models.GroupMember
		if err := db.Where("id = ? AND group_id = ?", input.FromMemberID, groupID).First(&fromMember).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Thành viên trả nợ không hợp lệ"})
			return
		}

		var toMember models.GroupMember
		if err := db.Where("id = ? AND group_id = ?", input.ToMemberID, groupID).First(&toMember).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Thành viên nhận tiền không hợp lệ"})
			return
		}

		settlement := models.Settlement{
			GroupID: uint(groupID),
			FromID:  input.FromMemberID,
			ToID:    input.ToMemberID,
			Amount:  input.Amount,
		}
		if err := db.Create(&settlement).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo bản ghi trả nợ"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "Xác nhận trả nợ thành công",
			"data":    settlement,
		})
	}
}
