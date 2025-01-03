package discord

import (
	"strconv"
	"time"
)

func parseSnowflakeToTime(snowflake string) (time.Time, error) {
	// Discord Snowflakes are based on Unix epoch time starting at 2015-01-01
	const discordEpoch int64 = 1420070400000 // Discord epoch in milliseconds

	// Convert the Snowflake to an integer
	id, err := strconv.ParseInt(snowflake, 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	// Extract the timestamp by shifting and adding the Discord epoch
	timestampMillis := (id >> 22) + discordEpoch
	return time.UnixMilli(timestampMillis), nil
}
