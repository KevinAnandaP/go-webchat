package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/vinneth/go-webchat/middleware"
	"github.com/vinneth/go-webchat/models"
	"github.com/vinneth/go-webchat/websocket"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CreateGroupRequest represents create group payload
type CreateGroupRequest struct {
	Name      string   `json:"name"`
	Icon      string   `json:"icon"`
	MemberIDs []string `json:"member_ids"`
}

// UpdateGroupRequest represents update group payload
type UpdateGroupRequest struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
}

// AddMemberRequest represents add member payload
type AddMemberRequest struct {
	UserID string `json:"user_id"`
}

// CreateGroup creates a new group
func CreateGroup(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req CreateGroupRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Group name is required",
		})
	}

	if len(req.MemberIDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "At least one member is required",
		})
	}

	// Convert member IDs
	memberIDs := make([]primitive.ObjectID, 0, len(req.MemberIDs))
	for _, idStr := range req.MemberIDs {
		id, err := primitive.ObjectIDFromHex(idStr)
		if err != nil {
			continue
		}
		// Verify member exists
		if _, err := models.FindUserByID(id); err == nil {
			memberIDs = append(memberIDs, id)
		}
	}

	// Create group
	group, err := models.CreateGroup(req.Name, req.Icon, userID, memberIDs)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create group",
		})
	}

	// Build response with member details
	membersList := make([]models.UserPublic, 0)
	for _, memberID := range group.Members {
		member, _ := models.FindUserByID(memberID)
		if member != nil {
			isOnline := websocket.Hub.IsOnline(member.ID)
			membersList = append(membersList, member.ToPublic(isOnline))
		}
	}

	result := models.ConversationWithDetails{
		Conversation: *group,
		MembersList:  membersList,
	}

	// Notify all members about new group
	for _, memberID := range group.Members {
		if memberID != userID {
			websocket.Hub.SendToUser(memberID, websocket.WSMessage{
				Type: "group:created",
				Payload: map[string]interface{}{
					"group": result,
				},
			})
		}
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"group": result,
	})
}

// UpdateGroup updates group info
func UpdateGroup(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	groupIDStr := c.Params("id")

	groupID, err := primitive.ObjectIDFromHex(groupIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid group ID",
		})
	}

	// Get group
	group, err := models.FindConversationByID(groupID)
	if err != nil || group == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Group not found",
		})
	}

	// Check if user is admin
	if group.Admin != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only admin can update group",
		})
	}

	var req UpdateGroupRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := models.UpdateGroup(groupID, req.Name, req.Icon); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update group",
		})
	}

	// Get updated group
	updatedGroup, _ := models.FindConversationByID(groupID)

	// Notify members
	for _, memberID := range group.Members {
		websocket.Hub.SendToUser(memberID, websocket.WSMessage{
			Type: "group:updated",
			Payload: map[string]interface{}{
				"group_id": groupID,
				"name":     updatedGroup.GroupName,
				"icon":     updatedGroup.GroupIcon,
			},
		})
	}

	return c.JSON(fiber.Map{
		"message": "Group updated successfully",
		"group":   updatedGroup,
	})
}

// AddGroupMember adds a member to group
func AddGroupMember(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	groupIDStr := c.Params("id")

	groupID, err := primitive.ObjectIDFromHex(groupIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid group ID",
		})
	}

	// Get group
	group, err := models.FindConversationByID(groupID)
	if err != nil || group == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Group not found",
		})
	}

	// Check if user is admin
	if group.Admin != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only admin can add members",
		})
	}

	var req AddMemberRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	memberID, err := primitive.ObjectIDFromHex(req.UserID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	// Check if user exists
	member, err := models.FindUserByID(memberID)
	if err != nil || member == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Add member
	if err := models.AddGroupMember(groupID, memberID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to add member",
		})
	}

	// Notify new member
	websocket.Hub.SendToUser(memberID, websocket.WSMessage{
		Type: "group:added",
		Payload: map[string]interface{}{
			"group_id":   groupID,
			"group_name": group.GroupName,
		},
	})

	// Notify existing members
	for _, existingMemberID := range group.Members {
		websocket.Hub.SendToUser(existingMemberID, websocket.WSMessage{
			Type: "group:member_added",
			Payload: map[string]interface{}{
				"group_id": groupID,
				"member":   member.ToPublic(websocket.Hub.IsOnline(memberID)),
			},
		})
	}

	return c.JSON(fiber.Map{
		"message": "Member added successfully",
	})
}

// RemoveGroupMember removes a member from group
func RemoveGroupMember(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	groupIDStr := c.Params("id")
	memberIDStr := c.Params("userId")

	groupID, err := primitive.ObjectIDFromHex(groupIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid group ID",
		})
	}

	memberID, err := primitive.ObjectIDFromHex(memberIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid member ID",
		})
	}

	// Get group
	group, err := models.FindConversationByID(groupID)
	if err != nil || group == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Group not found",
		})
	}

	// Check if user is admin or removing self
	if group.Admin != userID && memberID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only admin can remove other members",
		})
	}

	// Cannot remove admin
	if memberID == group.Admin {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot remove group admin",
		})
	}

	// Remove member
	if err := models.RemoveGroupMember(groupID, memberID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to remove member",
		})
	}

	// Notify removed member
	websocket.Hub.SendToUser(memberID, websocket.WSMessage{
		Type: "group:removed",
		Payload: map[string]interface{}{
			"group_id": groupID,
		},
	})

	// Notify remaining members
	for _, existingMemberID := range group.Members {
		if existingMemberID != memberID {
			websocket.Hub.SendToUser(existingMemberID, websocket.WSMessage{
				Type: "group:member_removed",
				Payload: map[string]interface{}{
					"group_id":  groupID,
					"member_id": memberID,
				},
			})
		}
	}

	return c.JSON(fiber.Map{
		"message": "Member removed successfully",
	})
}

// LeaveGroup allows a member to leave the group
func LeaveGroup(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	groupIDStr := c.Params("id")

	groupID, err := primitive.ObjectIDFromHex(groupIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid group ID",
		})
	}

	// Get group
	group, err := models.FindConversationByID(groupID)
	if err != nil || group == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Group not found",
		})
	}

	// Admin cannot leave (must assign new admin first)
	if group.Admin == userID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Admin cannot leave group. Transfer admin role first.",
		})
	}

	// Remove self
	if err := models.RemoveGroupMember(groupID, userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to leave group",
		})
	}

	// Notify other members
	for _, memberID := range group.Members {
		if memberID != userID {
			websocket.Hub.SendToUser(memberID, websocket.WSMessage{
				Type: "group:member_left",
				Payload: map[string]interface{}{
					"group_id":  groupID,
					"member_id": userID,
				},
			})
		}
	}

	return c.JSON(fiber.Map{
		"message": "Left group successfully",
	})
}
