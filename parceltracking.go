package main

import (
	"fmt"
	"strings"

	"github.com/osm/irc"
	"github.com/osm/pastebin"
	"github.com/osm/postnord"
)

// initParcelTrackingDefaults sets default values for all settings.
func (b *bot) initParcelTrackingDefaults() {
	if b.IRC.ParcelTrackingLocale == "" {
		b.IRC.ParcelTrackingLocale = "sv"
	}
	if b.IRC.ParcelTrackingMsgInfo == "" {
		b.IRC.ParcelTrackingMsgInfo = "<consignor_name>, <event_date> <event_time>, <location_display_name>, <event_description>"
	}
	if b.IRC.ParcelTrackingMsgAliasRemoved == "" {
		b.IRC.ParcelTrackingMsgAliasRemoved = "<alias> removed"
	}
	if b.IRC.ParcelTrackingMsgAliasDoesNotExist == "" {
		b.IRC.ParcelTrackingMsgAliasDoesNotExist = "<alias> does not exist"
	}
	if b.IRC.ParcelTrackingErrNoData == "" {
		b.IRC.ParcelTrackingErrNoData = "no tracking data found"
	}
	if b.IRC.ParcelTrackingErrDuplicateAlias == "" {
		b.IRC.ParcelTrackingErrDuplicateAlias = "<alias> is already in use for parcel <existing_id>"
	}
	if b.IRC.ParcelTrackingCmd == "" {
		b.IRC.ParcelTrackingCmd = "!pt"
	}
	if b.IRC.ParcelTrackingCmdAdd == "" {
		b.IRC.ParcelTrackingCmdAdd = "add"
	}
	if b.IRC.ParcelTrackingCmdRemove == "" {
		b.IRC.ParcelTrackingCmdRemove = "remove"
	}
	if b.IRC.ParcelTrackingCmdInfo == "" {
		b.IRC.ParcelTrackingCmdInfo = "info"
	}
	if b.IRC.ParcelTrackingCmdFull == "" {
		b.IRC.ParcelTrackingCmdFull = "full"
	}
}

// parcelTrackingCommandHandler handles the commands issued from the IRC channel.
func (b *bot) parcelTrackingCommandHandler(m *irc.Message) {
	a := b.parseAction(m).(*privmsgAction)

	// Not our channel, return.
	if !a.validChannel {
		return
	}

	// Not a parcel tracking command, return.
	if a.cmd != b.IRC.ParcelTrackingCmd {
		return
	}

	// Use should be ignored, return.
	if b.shouldIgnore(m) {
		return
	}

	// Not enough args, return.
	if len(a.args) < 2 {
		return
	}

	// Determine which sub command to execute.
	subCmd := a.args[0]
	if subCmd == b.IRC.ParcelTrackingCmdAdd {
		b.parcelTrackingAdd(a)
	} else if subCmd == b.IRC.ParcelTrackingCmdRemove {
		b.parcelTrackingRemove(a)
	} else if subCmd == b.IRC.ParcelTrackingCmdInfo {
		b.parcelTrackingInfo(a)
	} else if subCmd == b.IRC.ParcelTrackingCmdFull {
		b.parcelTrackingFull(a)
	}
}

// parcelTrackingAdd adds the given id and alias to the database and returns
// the latest event data to the channel.
func (b *bot) parcelTrackingAdd(a *privmsgAction) {
	// !pt add <id> <alias>
	if len(a.args) != 3 {
		return
	}

	// Store the arguments in convenient variable names.
	id := a.args[1]
	alias := a.args[2]

	// Make sure that the alias isn't used already.
	if existingID := b.parcelTrackingAliasExists(alias); existingID != "" {
		b.privmsgph(b.IRC.ParcelTrackingErrDuplicateAlias, map[string]string{
			"<alias>":       alias,
			"<existing_id>": existingID,
		})
		return

	}

	// Make sure that the requested id exists.
	events := b.fetchPostNordInfo(id)
	if events == nil {
		b.privmsg(b.IRC.ParcelTrackingErrNoData)
		return
	}

	// Store the id and alias.
	b.insertParcelTracking(alias, id)

	// Print the latest info
	b.sendParcelTrackingInfo(&events[len(events)-1])
}

// parcelTrackingRemove removes the alias from the database.
func (b *bot) parcelTrackingRemove(a *privmsgAction) {
	// !pt remove <alias>
	if len(a.args) != 2 {
		return
	}

	// Store the arguments in convenient variable names.
	alias := a.args[1]

	// Make sure that the ID exists before we try to remove it.
	if existingID := b.parcelTrackingAliasExists(alias); existingID == "" {
		b.privmsgph(b.IRC.ParcelTrackingMsgAliasDoesNotExist, map[string]string{
			"<alias>": alias,
		})
		return

	}

	// Delete it.
	stmt, err := b.prepare("UPDATE parcel_tracking SET is_deleted = 1 WHERE alias = ?")
	if err != nil {
		b.logger.Printf("parcelTracking: %v", err)
		b.privmsg(b.DB.Err)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(alias)
	if err != nil {
		b.logger.Printf("parcelTracking: %v", err)
		b.privmsg(b.DB.Err)
		return
	}

	// Print it to the channel.
	b.privmsgph(b.IRC.ParcelTrackingMsgAliasRemoved, map[string]string{
		"<alias>": alias,
	})
}

// parcelTrackingInfo returns the latest info for the given parcel id.
func (b *bot) parcelTrackingInfo(a *privmsgAction) {
	events := b.fetchParcelTrackingInfo(a)
	if events == nil {
		return
	}

	b.sendParcelTrackingInfo(&events[len(events)-1])
}

// parcelTrackingFull fetches a full details info for the given parcel id.
func (b *bot) parcelTrackingFull(a *privmsgAction) {
	events := b.fetchParcelTrackingInfo(a)
	if events == nil {
		return
	}

	// Compile a string of all events.
	var fullMsg string
	for _, e := range events {
		data := map[string]string{
			"<consignor_name>":        e.consignorName,
			"<event_date>":            e.eventDate,
			"<event_time>":            e.eventTime,
			"<location_display_name>": e.locationDisplayName,
			"<event_description>":     e.eventDescription,
		}

		msg := b.IRC.ParcelTrackingMsgInfo
		for k, v := range data {
			msg = strings.ReplaceAll(msg, k, v)
		}

		fullMsg = fmt.Sprintf("%s%s\n", fullMsg, msg)
	}

	// Remove the trailing new line.
	fullMsg = fullMsg[0 : len(fullMsg)-1]

	// Make sure that we have the pastebin API key.
	if b.IRC.PastebinAPIKey == "" {
		b.logger.Printf("parcelTracking: you need to set a pastebin api key\n")
		return
	}
	pb := pastebin.New(b.IRC.PastebinAPIKey)

	// Upload it to pastebin and print the URL.
	var url string
	url, err := pb.NewPaste(fullMsg, a.args[0], pastebin.Unlisted, pastebin.TenMinutes)
	if err != nil {
		b.logger.Printf("parcelTracking: pastebin err: %v\n", err)
		return
	}

	b.privmsg(url)

}

// parcelTrackingInfo fetches tracking info for the given id.
func (b *bot) fetchParcelTrackingInfo(a *privmsgAction) []postNordEvent {
	// !pt info <id|alias> [alias]
	if len(a.args) < 2 {
		return nil
	}

	// Store the arguments in convenient variable names.
	id := a.args[1]

	// Check if the given ID is actually an alias that we have stored in
	// the database.
	existingID := b.parcelTrackingAliasExists(id)
	if existingID != "" {
		id = existingID
	}

	// Make sure that the requested id exists.
	events := b.fetchPostNordInfo(id)
	if events == nil {
		b.privmsg(b.IRC.ParcelTrackingErrNoData)
		return nil
	}

	// If the optional alias parameter is sent we'll also insert the
	// parcel with an alias, as long as the alias isn't actually an alias
	// already.
	alias := ""
	if len(a.args) >= 3 {
		alias = a.args[2]
	}
	if alias != "" && existingID == "" && b.parcelTrackingAliasExists(alias) == "" {
		// Store the id and alias.
		b.insertParcelTracking(alias, id)
	}

	return events
}

// parcelTrackingAliasExists checks whether or not the alias is used, if it is
// we'll return the parcel tracking ID, otherwise we'll return an empty
// string.
func (b *bot) parcelTrackingAliasExists(alias string) string {
	var existingID string
	b.queryRow("SELECT parcel_tracking_id FROM parcel_tracking WHERE alias = ? AND is_deleted = 0", alias).Scan(&existingID)
	return existingID
}

// sendParcelTrackingInfo sends the given event to the channel.
func (b *bot) sendParcelTrackingInfo(e *postNordEvent) {
	b.privmsgph(b.IRC.ParcelTrackingMsgInfo, map[string]string{
		"<consignor_name>":        e.consignorName,
		"<event_date>":            e.eventDate,
		"<event_time>":            e.eventTime,
		"<location_display_name>": e.locationDisplayName,
		"<event_description>":     e.eventDescription,
	})
}

// insertParcelTracking adds an entry to the parcel_tracking table.
func (b *bot) insertParcelTracking(alias, id string) {
	stmt, err := b.prepare(`INSERT INTO parcel_tracking (
		id,
		alias,
		parcel_tracking_id,
		inserted_at,
		is_deleted
	) VALUES (
		?,
		?,
		?,
		?,
		?
	);`)
	if err != nil {
		b.logger.Printf("parcelTracking: %v", err)
		b.privmsg(b.DB.Err)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(newUUID(), alias, id, newTimestamp(), false)
	if err != nil {
		b.logger.Printf("parcelTracking: %v", err)
		b.privmsg(b.DB.Err)
		return
	}
}

// fetchPostNordInfo fetches info from the PostNord API for the given id.
func (b *bot) fetchPostNordInfo(id string) []postNordEvent {
	// Fetch the PostNord info.
	pn := postnord.New(b.IRC.ParcelTrackingPostNordAPIKey, b.IRC.ParcelTrackingLocale)
	tir, err := pn.FindByIdentifierV5(id)
	if err != nil {
		return nil
	}

	// PostNord does not return 404 if the id is incorrect, so we'll just
	// assume it's incorrect if the response doesn't have any shipments.
	if len(tir.TrackingInformationResponse.Shipments) == 0 {
		return nil
	}

	// Convert the PostNord API data into our local data structure.
	var events []postNordEvent
	for _, s := range tir.TrackingInformationResponse.Shipments {
		for _, i := range s.Items {
			for _, e := range i.Events {
				events = append(events, postNordEvent{
					consignorName:       s.Consignor.Name,
					eventDate:           e.EventTime[0:10],
					eventTime:           e.EventTime[11:16],
					locationDisplayName: e.Location.DisplayName,
					eventDescription:    e.EventDescription,
				})
			}
		}
	}

	return events
}

// postNordEvent simplified data structure for the "important" PostNord event
// data.
type postNordEvent struct {
	consignorName       string
	eventDate           string
	eventTime           string
	locationDisplayName string
	eventDescription    string
}
