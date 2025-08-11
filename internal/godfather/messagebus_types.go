package godfather

type AlertMessage struct {
	Subject        string `msgpack:"subject"`
	NotificationId int    `msgpack:"notification_id"`
}
