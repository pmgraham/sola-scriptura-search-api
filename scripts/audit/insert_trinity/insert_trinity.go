package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"github.com/lib/pq"
)

// CanonicalVerse represents a verse that should be in a topic
type CanonicalVerse struct {
	VerseID    string
	Importance int    // 1 = essential, 2 = important, 3 = supporting
	Reason     string // Why this verse is relevant to the topic
}

// TrinityCanonicalVerses - the verses that SHOULD be in the Trinity topic
var TrinityCanonicalVerses = []CanonicalVerse{
	// Tier 1: Essential
	{VerseID: "Matt.28.19", Importance: 1, Reason: "Great Commission - baptize in name of Father, Son, Holy Spirit"},
	{VerseID: "2Cor.13.14", Importance: 1, Reason: "Apostolic benediction - grace of Christ, love of God, fellowship of Spirit"},
	{VerseID: "John.1.1", Importance: 1, Reason: "The Word was God - deity of Christ"},
	{VerseID: "John.1.14", Importance: 1, Reason: "The Word became flesh - incarnation"},
	{VerseID: "Gen.1.26", Importance: 1, Reason: "Let Us make man - plural divine council"},
	{VerseID: "John.10.30", Importance: 1, Reason: "I and the Father are one"},
	{VerseID: "Col.2.9", Importance: 1, Reason: "Fullness of deity dwells bodily in Christ"},
	{VerseID: "Isa.9.6", Importance: 1, Reason: "Mighty God, Everlasting Father - messianic prophecy"},
	{VerseID: "John.20.28", Importance: 1, Reason: "Thomas: My Lord and my God"},
	{VerseID: "Heb.1.3", Importance: 1, Reason: "Radiance of God's glory, exact imprint"},

	// Tier 2: Important
	{VerseID: "Phil.2.6", Importance: 2, Reason: "Christ existed in form of God"},
	{VerseID: "Phil.2.7", Importance: 2, Reason: "Emptied himself, took form of servant"},
	{VerseID: "John.8.58", Importance: 2, Reason: "Before Abraham was, I AM"},
	{VerseID: "John.14.9", Importance: 2, Reason: "Whoever has seen me has seen the Father"},
	{VerseID: "John.14.10", Importance: 2, Reason: "I am in the Father and the Father is in me"},
	{VerseID: "Titus.2.13", Importance: 2, Reason: "Our great God and Savior Jesus Christ"},
	{VerseID: "Rom.9.5", Importance: 2, Reason: "Christ who is God over all"},
	{VerseID: "Acts.5.3", Importance: 2, Reason: "Lying to Holy Spirit = lying to God"},
	{VerseID: "Acts.5.4", Importance: 2, Reason: "You have not lied to man but to God"},
	{VerseID: "1John.5.7", Importance: 2, Reason: "Three that bear witness (Johannine Comma)"},
	{VerseID: "Isa.6.8", Importance: 2, Reason: "Whom shall I send, who will go for Us?"},
	{VerseID: "Gen.1.1", Importance: 2, Reason: "In the beginning God (Elohim - plural)"},
	{VerseID: "Gen.1.2", Importance: 2, Reason: "Spirit of God hovering over waters"},
	{VerseID: "Gen.11.7", Importance: 2, Reason: "Let Us go down - Tower of Babel"},

	// Tier 3: Supporting
	{VerseID: "John.14.16", Importance: 3, Reason: "Father will give another Helper"},
	{VerseID: "John.14.26", Importance: 3, Reason: "Holy Spirit whom the Father will send"},
	{VerseID: "John.15.26", Importance: 3, Reason: "Spirit of truth who proceeds from Father"},
	{VerseID: "John.16.13", Importance: 3, Reason: "Spirit of truth will guide you"},
	{VerseID: "Matt.3.16", Importance: 3, Reason: "Baptism of Jesus - all three persons present"},
	{VerseID: "Matt.3.17", Importance: 3, Reason: "Voice from heaven: This is my beloved Son"},
	{VerseID: "Luke.1.35", Importance: 3, Reason: "Holy Spirit will come upon you"},
	{VerseID: "John.5.18", Importance: 3, Reason: "Making himself equal with God"},
	{VerseID: "John.17.5", Importance: 3, Reason: "Glory I had with you before the world existed"},
	{VerseID: "John.17.21", Importance: 3, Reason: "That they may be one as we are one"},
	{VerseID: "1Cor.8.6", Importance: 3, Reason: "One God the Father, one Lord Jesus Christ"},
	{VerseID: "1Cor.12.4", Importance: 3, Reason: "Same Spirit, same Lord, same God"},
	{VerseID: "1Cor.12.5", Importance: 3, Reason: "Varieties of service, same Lord"},
	{VerseID: "1Cor.12.6", Importance: 3, Reason: "Same God who empowers them all"},
	{VerseID: "Eph.4.4", Importance: 3, Reason: "One Spirit"},
	{VerseID: "Eph.4.5", Importance: 3, Reason: "One Lord"},
	{VerseID: "Eph.4.6", Importance: 3, Reason: "One God and Father of all"},
	{VerseID: "1Pet.1.2", Importance: 3, Reason: "Foreknowledge of Father, sanctification of Spirit, sprinkling of blood of Jesus"},
	{VerseID: "Rev.1.4", Importance: 3, Reason: "Grace from him who is, seven spirits, Jesus Christ"},
	{VerseID: "Rev.1.8", Importance: 3, Reason: "I am the Alpha and Omega - applied to Christ"},
}

func main() {
	godotenv.Load()

	db, err := sqlx.Connect("postgres", os.Getenv("POSTGRES_URI"))
	if err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	ctx := context.Background()

	// Start transaction
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		fmt.Printf("Failed to begin transaction: %v\n", err)
		os.Exit(1)
	}
	defer tx.Rollback()

	// Insert the Trinity topic with claude_4.5_opus source
	var topicID int
	insertTopic := `
		INSERT INTO api.topics (name, slug, source, topic, sub_topic, category, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	err = tx.QueryRowContext(ctx, insertTopic,
		"Trinity",                                    // name
		"trinity",                                    // slug
		"claude_4.5_opus",                            // source
		"Trinity",                                    // topic
		"",                                           // sub_topic
		"concept",                                    // category
		"The doctrine of the Trinity: One God existing eternally as three distinct persons - Father, Son, and Holy Spirit. This collection includes verses demonstrating the deity of each person and their unity.", // description
	).Scan(&topicID)
	if err != nil {
		fmt.Printf("Failed to insert topic: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Created topic with ID: %d\n", topicID)

	// Get verse IDs for all canonical verses
	verseIDs := make([]string, len(TrinityCanonicalVerses))
	for i, cv := range TrinityCanonicalVerses {
		verseIDs[i] = cv.VerseID
	}

	// Query to get verse numeric IDs
	query := `SELECT id, osis_verse_id FROM api.verses WHERE osis_verse_id = ANY($1)`
	rows, err := tx.QueryContext(ctx, query, pq.Array(verseIDs))
	if err != nil {
		fmt.Printf("Failed to query verses: %v\n", err)
		os.Exit(1)
	}

	verseIDMap := make(map[string]int)
	for rows.Next() {
		var id int
		var osisID string
		if err := rows.Scan(&id, &osisID); err != nil {
			fmt.Printf("Failed to scan verse: %v\n", err)
			os.Exit(1)
		}
		verseIDMap[osisID] = id
	}
	rows.Close()

	fmt.Printf("Found %d verses in database\n", len(verseIDMap))

	// Insert topic_verses mappings
	insertMapping := `INSERT INTO api.topic_verses (topic_id, verse_id) VALUES ($1, $2)`
	insertedCount := 0
	missingVerses := []string{}

	for _, cv := range TrinityCanonicalVerses {
		verseID, ok := verseIDMap[cv.VerseID]
		if !ok {
			missingVerses = append(missingVerses, cv.VerseID)
			continue
		}

		_, err := tx.ExecContext(ctx, insertMapping, topicID, verseID)
		if err != nil {
			fmt.Printf("Failed to insert mapping for %s: %v\n", cv.VerseID, err)
			continue
		}
		insertedCount++
	}

	if len(missingVerses) > 0 {
		fmt.Printf("\nWarning: %d verses not found in database:\n", len(missingVerses))
		for _, v := range missingVerses {
			fmt.Printf("  - %s\n", v)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		fmt.Printf("Failed to commit transaction: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✅ Successfully created Trinity topic (ID: %d) with %d verses\n", topicID, insertedCount)
	fmt.Println("   Source: claude_4.5_opus")
}
