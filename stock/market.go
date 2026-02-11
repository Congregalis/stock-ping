package stock

import (
	"time"
)

// Market constants
const (
	MarketUS     = "US"
	MarketCN     = "CN"
	MarketHK     = "HK"
	MarketTW     = "TW"
	MarketCrypto = "CRYPTO"
	MarketForex  = "FOREX"
)

// Timezones
var (
	tzEastern  *time.Location
	tzShanghai *time.Location
)

func init() {
	var err error
	tzEastern, err = time.LoadLocation("America/New_York")
	if err != nil {
		tzEastern = time.FixedZone("EST", -5*60*60)
	}

	tzShanghai, err = time.LoadLocation("Asia/Shanghai")
	if err != nil {
		tzShanghai = time.FixedZone("CST", 8*60*60)
	}
}

// IsMarketOpen checks if a specific market is currently open
func IsMarketOpen(market string) bool {
	switch market {
	case MarketCrypto:
		return true // Crypto is 24/7
	case MarketCN:
		return isCNMarketOpen()
	case MarketTW:
		return isTWMarketOpen()
	case MarketForex:
		return isForexOpen()
	case MarketUS:
		return isUSMarketOpen()
	default:
		// Default to US if not specified or unknown
		return isUSMarketOpen()
	}
}

func isUSMarketOpen() bool {
	now := time.Now().In(tzEastern)
	weekday := now.Weekday()

	// Weekends
	if weekday == time.Saturday || weekday == time.Sunday {
		return false
	}

	hour, min, _ := now.Clock()
	timeInMinutes := hour*60 + min

	// 09:30 - 16:00
	return timeInMinutes >= 9*60+30 && timeInMinutes < 16*60
}

func isCNMarketOpen() bool {
	now := time.Now().In(tzShanghai)
	weekday := now.Weekday()

	// Weekends
	if weekday == time.Saturday || weekday == time.Sunday {
		return false
	}

	hour, min, _ := now.Clock()
	timeInMinutes := hour*60 + min

	// Morning: 09:30 - 11:30
	// Afternoon: 13:00 - 15:00
	isMorning := timeInMinutes >= 9*60+30 && timeInMinutes < 11*60+30
	isAfternoon := timeInMinutes >= 13*60 && timeInMinutes < 15*60

	return isMorning || isAfternoon
}

func isTWMarketOpen() bool {
	// Taiwan shares timezone with Shanghai (CST/GMT+8)
	now := time.Now().In(tzShanghai)
	weekday := now.Weekday()

	// Weekends
	if weekday == time.Saturday || weekday == time.Sunday {
		return false
	}

	hour, min, _ := now.Clock()
	timeInMinutes := hour*60 + min

	// 09:00 - 13:30 (No lunch break)
	return timeInMinutes >= 9*60 && timeInMinutes < 13*60+30
}

func isForexOpen() bool {
	// Forex/Metals typically trade 24/5
	// Opens Sunday 17:00 EST, Closes Friday 17:00 EST
	now := time.Now().In(tzEastern)
	weekday := now.Weekday()

	if weekday == time.Saturday {
		return false
	}

	hour, _, _ := now.Clock()

	// Friday: Close after 17:00
	if weekday == time.Friday && hour >= 17 {
		return false
	}

	// Sunday: Open after 17:00
	if weekday == time.Sunday && hour < 17 {
		return false
	}

	return true
}

// GetNextMarketOpen returns the next opening time for the given market
// This is used for UI countdowns (optional)
func GetNextMarketOpen(market string) time.Time {
	switch market {
	case MarketCrypto:
		return time.Now()
	case MarketForex:
		return getNextForexOpen()
	case MarketCN:
		return getNextCNMarketOpen()
	case MarketTW:
		return getNextTWMarketOpen()
	default:
		return getNextUSMarketOpen()
	}
}

func getNextUSMarketOpen() time.Time {
	now := time.Now().In(tzEastern)

	// Create today's open time
	nextOpen := time.Date(now.Year(), now.Month(), now.Day(), 9, 30, 0, 0, tzEastern)

	// If already past open time today, move to tomorrow
	if now.After(nextOpen) {
		nextOpen = nextOpen.Add(24 * time.Hour)
	}

	// Skip weekends
	for nextOpen.Weekday() == time.Saturday || nextOpen.Weekday() == time.Sunday {
		nextOpen = nextOpen.Add(24 * time.Hour)
	}

	return nextOpen
}

func getNextCNMarketOpen() time.Time {
	now := time.Now().In(tzShanghai)

	// Morning open
	morningOpen := time.Date(now.Year(), now.Month(), now.Day(), 9, 30, 0, 0, tzShanghai)
	// Afternoon open
	afternoonOpen := time.Date(now.Year(), now.Month(), now.Day(), 13, 0, 0, 0, tzShanghai)

	if now.Before(morningOpen) {
		return morningOpen
	}

	if now.Before(afternoonOpen) {
		// If currently in lunch break (11:30 - 13:00), next is afternoon open
		hour, min, _ := now.Clock()
		mins := hour*60 + min

		if mins < 11*60+30 {
			// It is open, so... invalid call or return now?
			return now
		}
		// Lunch break
		return afternoonOpen
	}

	// After afternoon open.
	// If in afternoon session, it's open.

	// If closed for the day (after 15:00), next is tomorrow 9:30
	nextOpen := morningOpen.Add(24 * time.Hour)

	// Skip weekends
	for nextOpen.Weekday() == time.Saturday || nextOpen.Weekday() == time.Sunday {
		nextOpen = nextOpen.Add(24 * time.Hour)
	}

	return nextOpen
}

func getNextTWMarketOpen() time.Time {
	now := time.Now().In(tzShanghai)

	// Open at 09:00
	nextOpen := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, tzShanghai)

	if now.After(nextOpen) {
		nextOpen = nextOpen.Add(24 * time.Hour)
	}

	// Skip weekends
	for nextOpen.Weekday() == time.Saturday || nextOpen.Weekday() == time.Sunday {
		nextOpen = nextOpen.Add(24 * time.Hour)
	}

	return nextOpen
}

func getNextForexOpen() time.Time {
	now := time.Now().In(tzEastern)
	// Opens Sunday 17:00

	// Find next Sunday
	daysUntilSunday := (int(time.Sunday) - int(now.Weekday()) + 7) % 7
	if daysUntilSunday == 0 && now.Hour() >= 17 {
		// Already open or past open on Sunday
		daysUntilSunday = 7
	}

	nextOpen := time.Date(now.Year(), now.Month(), now.Day(), 17, 0, 0, 0, tzEastern)
	nextOpen = nextOpen.Add(time.Duration(daysUntilSunday) * 24 * time.Hour)

	return nextOpen
}
