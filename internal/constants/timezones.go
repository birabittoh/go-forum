package constants

type Timezone struct {
	Name  string
	Value string
}

// Helper to get all timezones
func TimezonesList() []Timezone {
	return []Timezone{
		{Name: "GMT-12:00 Baker Island", Value: "Etc/GMT+12"},
		{Name: "GMT-11:00 Pago Pago", Value: "Pacific/Pago_Pago"},
		{Name: "GMT-10:00 Honolulu", Value: "Pacific/Honolulu"},
		{Name: "GMT-09:00 Anchorage", Value: "America/Anchorage"},
		{Name: "GMT-08:00 Los Angeles, Vancouver", Value: "America/Los_Angeles"},
		{Name: "GMT-07:00 Denver, Phoenix", Value: "America/Denver"},
		{Name: "GMT-06:00 Chicago, Mexico City", Value: "America/Chicago"},
		{Name: "GMT-05:00 New York, Toronto, Lima", Value: "America/New_York"},
		{Name: "GMT-04:00 Santiago, Caracas", Value: "America/Santiago"},
		{Name: "GMT-03:30 St. John's", Value: "America/St_Johns"},
		{Name: "GMT-03:00 Buenos Aires, São Paulo", Value: "America/Sao_Paulo"},
		{Name: "GMT-02:00 South Georgia", Value: "Atlantic/South_Georgia"},
		{Name: "GMT-01:00 Azores, Cape Verde", Value: "Atlantic/Azores"},
		{Name: "GMT+00:00 London, Lisbon, Reykjavik", Value: "UTC"},
		{Name: "GMT+01:00 Paris, Berlin, Rome, Madrid", Value: "Europe/Rome"},
		{Name: "GMT+02:00 Athens, Cairo, Jerusalem", Value: "Europe/Athens"},
		{Name: "GMT+03:00 Moscow, Istanbul, Nairobi", Value: "Europe/Moscow"},
		{Name: "GMT+03:30 Tehran", Value: "Asia/Tehran"},
		{Name: "GMT+04:00 Dubai, Baku", Value: "Asia/Dubai"},
		{Name: "GMT+04:30 Kabul", Value: "Asia/Kabul"},
		{Name: "GMT+05:00 Karachi, Tashkent", Value: "Asia/Karachi"},
		{Name: "GMT+05:30 Mumbai, Delhi, Colombo", Value: "Asia/Kolkata"},
		{Name: "GMT+05:45 Kathmandu", Value: "Asia/Kathmandu"},
		{Name: "GMT+06:00 Dhaka, Almaty", Value: "Asia/Dhaka"},
		{Name: "GMT+06:30 Yangon", Value: "Asia/Yangon"},
		{Name: "GMT+07:00 Bangkok, Jakarta, Hanoi", Value: "Asia/Bangkok"},
		{Name: "GMT+08:00 Beijing, Singapore, Hong Kong", Value: "Asia/Shanghai"},
		{Name: "GMT+08:45 Eucla", Value: "Australia/Eucla"},
		{Name: "GMT+09:00 Tokyo, Seoul", Value: "Asia/Tokyo"},
		{Name: "GMT+09:30 Adelaide, Darwin", Value: "Australia/Adelaide"},
		{Name: "GMT+10:00 Sydney, Melbourne, Brisbane", Value: "Australia/Sydney"},
		{Name: "GMT+10:30 Lord Howe Island", Value: "Australia/Lord_Howe"},
		{Name: "GMT+11:00 Noumea, Solomon Islands", Value: "Pacific/Guadalcanal"},
		{Name: "GMT+12:00 Auckland, Fiji", Value: "Pacific/Auckland"},
		{Name: "GMT+12:45 Chatham Islands", Value: "Pacific/Chatham"},
		{Name: "GMT+13:00 Apia, Nuku'alofa", Value: "Pacific/Apia"},
		{Name: "GMT+14:00 Kiritimati", Value: "Pacific/Kiritimati"},
	}
}
