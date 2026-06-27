package controllers

import (
	"expensetracker/models"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CreateGroupInput struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Type        string `json:"type" binding:"omitempty,oneof=regular peer_to_peer"`
}

type AddMemberInput struct {
	Email     string `json:"email" binding:"omitempty,email"`
	GuestName string `json:"guest_name"`
}

func CreateGroup(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("currentUserID").(uint)

		var input CreateGroupInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ", "details": err.Error()})
			return
		}

		// Default type to "regular" if not provided
		groupType := input.Type
		if groupType == "" {
			groupType = "regular"
		}

		tx := db.Begin()

		group := models.Group{
			Name:        input.Name,
			Description: input.Description,
			Type:        groupType,
			CreatedBy:   userID,
		}
		if err := tx.Create(&group).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo nhóm"})
			return
		}

		member := models.GroupMember{
			GroupID: group.ID,
			UserID:  &userID,
			Role:    "admin",
		}
		if err := tx.Create(&member).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể thêm thành viên"})
			return
		}

		tx.Commit()

		c.JSON(http.StatusCreated, gin.H{
			"message": "Tạo nhóm thành công",
			"data":    group,
		})
	}
}

func GetGroups(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("currentUserID").(uint)

		var groups []models.Group
		err := db.Raw(`
			SELECT g.* FROM groups g
			INNER JOIN group_members gm ON gm.group_id = g.id
			WHERE gm.user_id = ? AND g.deleted_at IS NULL
			GROUP BY g.id
			ORDER BY g.created_at DESC
		`, userID).Scan(&groups).Error
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy danh sách nhóm"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": groups})
	}
}

func GetGroupDetails(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("currentUserID").(uint)
		groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID nhóm không hợp lệ"})
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

		var members []models.GroupMember
		db.Where("group_id = ?", group.ID).Preload("User").Find(&members)

		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"group":   group,
				"members": members,
			},
		})
	}
}

type UpdateGroupInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func UpdateGroup(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("currentUserID").(uint)
		groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID nhóm không hợp lệ"})
			return
		}

		var group models.Group
		if err := db.First(&group, groupID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy nhóm"})
			return
		}

		if group.CreatedBy != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ admin mới có quyền sửa nhóm"})
			return
		}

		var input UpdateGroupInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ", "details": err.Error()})
			return
		}

		updates := map[string]interface{}{}
		if input.Name != "" {
			updates["name"] = input.Name
		}
		if input.Description != "" {
			updates["description"] = input.Description
		}

		if len(updates) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không có dữ liệu cần cập nhật"})
			return
		}

		if err := db.Model(&group).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật nhóm"})
			return
		}

		db.First(&group, groupID)

		c.JSON(http.StatusOK, gin.H{
			"message": "Cập nhật nhóm thành công",
			"data":    group,
		})
	}
}

func AddMember(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("currentUserID").(uint)
		groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID nhóm không hợp lệ"})
			return
		}

		var group models.Group
		if err := db.First(&group, groupID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy nhóm"})
			return
		}

		if group.CreatedBy != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ admin mới có quyền mời thành viên"})
			return
		}

		var input AddMemberInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ", "details": err.Error()})
			return
		}

		if input.Email == "" && input.GuestName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Phải cung cấp email hoặc guest_name"})
			return
		}

		var memberUser *models.User
		if input.Email != "" {
			var foundUser models.User
			if err := db.Where("email = ?", input.Email).First(&foundUser).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người dùng với email này"})
				return
			}
			memberUser = &foundUser

			var count int64
			db.Model(&models.GroupMember{}).Where("group_id = ? AND user_id = ?", group.ID, foundUser.ID).Count(&count)
			if count > 0 {
				c.JSON(http.StatusConflict, gin.H{"error": "Người dùng đã là thành viên của nhóm"})
				return
			}
		}

		var member models.GroupMember
		if memberUser != nil {
			member = models.GroupMember{
				GroupID: group.ID,
				UserID:  &memberUser.ID,
				Role:    "member",
			}
		} else {
			member = models.GroupMember{
				GroupID:   group.ID,
				GuestName: input.GuestName,
				Role:      "member",
			}
		}
		if err := db.Create(&member).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể thêm thành viên"})
			return
		}

		if NotifSvc != nil && memberUser != nil && memberUser.Email != "" {
			var creator models.User
			db.First(&creator, userID)

			NotifSvc.CreateAndDispatch(
				memberUser.ID,
				"group_invite",
				"Bạn được mời vào nhóm",
				fmt.Sprintf("%s đã mời bạn vào nhóm \"%s\"", creator.Username, group.Name),
				nil,
				true,
				memberUser.Email,
				fmt.Sprintf("Bạn được mời vào nhóm %s", group.Name),
				fmt.Sprintf("%s đã mời bạn vào nhóm \"%s\". Hãy mở ứng dụng để xem chi tiết.", creator.Username, group.Name),
			)
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "Thêm thành viên thành công",
			"data":    member,
		})
	}
}

func RemoveMember(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("currentUserID").(uint)
		groupID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
		memberID, _ := strconv.ParseUint(c.Param("member_id"), 10, 32)

		var group models.Group
		if err := db.First(&group, groupID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy nhóm"})
			return
		}

		if group.CreatedBy != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ admin mới có quyền xóa thành viên"})
			return
		}

		var member models.GroupMember
		if err := db.Where("id = ? AND group_id = ?", memberID, group.ID).First(&member).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy thành viên"})
			return
		}

		if member.UserID != nil && *member.UserID == userID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể xóa chính mình"})
			return
		}

		if err := db.Delete(&member).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa thành viên"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Xóa thành viên thành công"})
	}
}

func RemindDebt(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("currentUserID").(uint)
		groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID nhóm không hợp lệ"})
			return
		}
		memberID, err := strconv.ParseUint(c.Param("member_id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID thành viên không hợp lệ"})
			return
		}

		var isMember int64
		db.Model(&models.GroupMember{}).Where("group_id = ? AND user_id = ?", groupID, userID).Count(&isMember)
		if isMember == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "Bạn không phải thành viên của nhóm này"})
			return
		}

		var toMember models.GroupMember
		if err := db.Where("id = ? AND group_id = ?", memberID, groupID).First(&toMember).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy thành viên"})
			return
		}

		if toMember.UserID == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Thành viên này là khách, không có email để nhắc"})
			return
		}

		var toUser models.User
		db.First(&toUser, *toMember.UserID)

		type debtResult struct {
			SharedBillID uint   `json:"shared_bill_id"`
			Amount       int    `json:"amount"`
			Description  string `json:"description"`
			GroupName    string `json:"group_name"`
		}
		var debts []debtResult
		db.Raw(`
			SELECT bs.id AS shared_bill_id, bs.amount, sb.description, g.name AS group_name
			FROM bill_splits bs
			JOIN shared_bills sb ON sb.id = bs.shared_bill_id
			JOIN groups g ON g.id = sb.group_id
			WHERE sb.group_id = ?
			  AND bs.group_member_id = ?
			  AND bs.is_settled = false
			  AND sb.payer_id != ?
			  AND bs.deleted_at IS NULL
			  AND sb.deleted_at IS NULL
		`, groupID, memberID, memberID).Scan(&debts)

		if len(debts) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Thành viên này không nợ tiền"})
			return
		}

		today := time.Now().Truncate(24 * time.Hour)
		var todayReminders int64
		db.Model(&models.DebtReminder{}).
			Where("group_id = ? AND to_member_id = ? AND reminder_type = ? AND sent_at >= ?",
				groupID, memberID, "manual", today).
			Count(&todayReminders)
		if todayReminders > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đã nhắc thành viên này hôm nay rồi"})
			return
		}

		totalOwed := 0
		billDescs := ""
		for i, d := range debts {
			totalOwed += d.Amount
			if i < 3 {
				if i > 0 {
					billDescs += ", "
				}
				billDescs += d.Description
			}
		}
		if len(debts) > 3 {
			billDescs += fmt.Sprintf(" và %d hóa đơn khác", len(debts)-3)
		}

		var fromMember models.GroupMember
		db.Where("group_id = ? AND user_id = ?", groupID, userID).First(&fromMember)

		for _, d := range debts {
			reminder := models.DebtReminder{
				GroupID:      uint(groupID),
				SharedBillID: d.SharedBillID,
				FromMemberID: fromMember.ID,
				ToMemberID:   toMember.ID,
				ReminderType: "manual",
			}
			db.Create(&reminder)
		}

		if NotifSvc != nil {
			var group models.Group
			db.First(&group, groupID)

			NotifSvc.CreateAndDispatch(
				*toMember.UserID,
				"debt_reminder",
				"Nhắc nhở thanh toán",
				fmt.Sprintf("Bạn nợ %d VND trong nhóm \"%s\" từ các hóa đơn: %s",
					totalOwed, group.Name, billDescs),
				nil,
				true,
				toUser.Email,
				fmt.Sprintf("Nhắc nhở: Bạn có khoản nợ trong nhóm %s", group.Name),
				fmt.Sprintf("Bạn đang nợ %d VND trong nhóm \"%s\" từ các hóa đơn: %s. Hãy thanh toán sớm nhé!",
					totalOwed, group.Name, billDescs),
			)
		}

		c.JSON(http.StatusOK, gin.H{
			"message":    "Đã gửi nhắc nhở thành công",
			"total_owed": totalOwed,
			"debts":      debts,
		})
	}
}
