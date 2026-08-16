package main

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

const discordLeaderboardLimit = 10

var discordEmojiPattern = regexp.MustCompile(`^<(a?):([A-Za-z0-9_]+):(\d{17,20})>?$`)

func buildLeaderboardEmbed(guildID string, guildName string, things []PointItem, members []PointItem, public bool, author *discordgo.User) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Type:        discordgo.EmbedTypeRich,
		Title:       "DuckBot Leaderboard",
		Description: fmt.Sprintf("Top point earners in **%s**", escapeDiscordMarkdown(guildName)),
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Color:       0x0E6E66,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Top Things",
				Value:  formatLeaderboardEntries(things, false),
				Inline: true,
			},
			{
				Name:   "Top Members",
				Value:  formatLeaderboardEntries(members, true),
				Inline: true,
			},
		},
	}

	if author != nil {
		embed.Footer = &discordgo.MessageEmbedFooter{
			IconURL: author.AvatarURL(""),
			Text:    fmt.Sprintf("Requested by %s", author.Username),
		}
	}
	if public {
		embed.URL = fmt.Sprintf("https://duckpoints.com/leaderboard.html?guild=%s", guildID)
		embed.Description += "\n[View the full leaderboard](" + embed.URL + ")"
	}
	return embed
}

func formatLeaderboardEntries(entries []PointItem, members bool) string {
	if len(entries) == 0 {
		return "_No points yet._"
	}

	limit := len(entries)
	if limit > discordLeaderboardLimit {
		limit = discordLeaderboardLimit
	}

	lines := make([]string, 0, limit)
	for index, entry := range entries[:limit] {
		name := formatLeaderboardName(entry.Item, members)
		lines = append(lines, fmt.Sprintf("**%d.** %s — **%s**", index+1, name, formatPointTotal(entry.Points)))
	}
	return strings.Join(lines, "\n")
}

func formatLeaderboardName(name string, member bool) string {
	if member && isDiscordGuildID(name) {
		return "<@" + name + ">"
	}

	if match := discordEmojiPattern.FindStringSubmatch(name); match != nil {
		return fmt.Sprintf("<%s:%s:%s>", match[1], match[2], match[3])
	}

	return escapeDiscordMarkdown(truncateRunes(name, 36))
}

func formatPointTotal(points float64) string {
	if math.Trunc(points) != points {
		return strconv.FormatFloat(points, 'f', 2, 64)
	}

	value := strconv.FormatInt(int64(points), 10)
	sign := ""
	if strings.HasPrefix(value, "-") {
		sign = "-"
		value = strings.TrimPrefix(value, "-")
	}

	for position := len(value) - 3; position > 0; position -= 3 {
		value = value[:position] + "," + value[position:]
	}
	return sign + value
}

func escapeDiscordMarkdown(value string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"*", "\\*",
		"_", "\\_",
		"~", "\\~",
		"`", "\\`",
		"|", "\\|",
	)
	return replacer.Replace(value)
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit-1]) + "..."
}
