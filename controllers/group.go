package controllers

import (
	"expensetracker/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CreateGroupInput struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Type        string `json:"type" binding:"omitempty,oneof=regular peer_to_peer"` // Default: regular
}

type AddMemberInput struct {
	UserID    *uint  `json:"user_id"`
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

		if input.UserID == nil && input.GuestName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Phải cung cấp user_id hoặc guest_name"})
			return
		}

		if input.UserID != nil {
			var count int64
			db.Model(&models.GroupMember{}).Where("group_id = ? AND user_id = ?", group.ID, *input.UserID).Count(&count)
			if count > 0 {
				c.JSON(http.StatusConflict, gin.H{"error": "Người dùng đã là thành viên của nhóm"})
				return
			}
		}

		member := models.GroupMember{
			GroupID:   group.ID,
			UserID:    input.UserID,
			GuestName: input.GuestName,
			Role:      "member",
		}
		if err := db.Create(&member).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể thêm thành viên"})
			return
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
