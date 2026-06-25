package controllers

import "expensetracker/services"

// NotificationService là global reference để sử dụng trong controllers
var NotifSvc *services.NotificationService

func SetNotificationService(svc *services.NotificationService) {
	NotifSvc = svc
}
