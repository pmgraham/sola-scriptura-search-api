package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// CanonicalVerse represents a verse that should be in a topic
type CanonicalVerse struct {
	VerseID    string
	Importance int    // 1 = essential, 2 = important, 3 = supporting
	Reason     string // Why this verse is relevant to the topic
}

// TrinityCanonicalVerses - the verses that SHOULD be in the Trinity topic
// Based on theological consensus across traditions
var TrinityCanonicalVerses = []CanonicalVerse{
	// Tier 1: Essential - These are the definitive Trinity proof texts
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

	// Tier 2: Important - Strong supporting texts
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

	// Tier 3: Supporting - Verses that illuminate the doctrine
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

	// Get all verses currently in Trinity topic (33509)
	query := `
		SELECT v.osis_verse_id
		FROM api.topic_verses tv
		JOIN api.verses v ON tv.verse_id = v.id
		WHERE tv.topic_id = 33509
	`
	var existingVerses []string
	if err := db.SelectContext(ctx, &existingVerses, query); err != nil {
		fmt.Printf("Failed to query existing verses: %v\n", err)
		os.Exit(1)
	}

	// Create a set for quick lookup
	existingSet := make(map[string]bool)
	for _, v := range existingVerses {
		existingSet[v] = true
	}

	fmt.Println("=" + strings.Repeat("=", 79))
	fmt.Println("TRINITY TOPIC AUDIT")
	fmt.Println("=" + strings.Repeat("=", 79))
	fmt.Printf("\nExisting verses in Trinity topic (33509): %d\n", len(existingVerses))
	fmt.Printf("Canonical verses defined: %d\n\n", len(TrinityCanonicalVerses))

	// Find missing verses
	var missing []CanonicalVerse
	var present []CanonicalVerse

	for _, cv := range TrinityCanonicalVerses {
		if existingSet[cv.VerseID] {
			present = append(present, cv)
		} else {
			missing = append(missing, cv)
		}
	}

	// Report missing by tier
	fmt.Println("-" + strings.Repeat("-", 79))
	fmt.Println("MISSING VERSES (should be added)")
	fmt.Println("-" + strings.Repeat("-", 79))

	tier1Missing := filterByImportance(missing, 1)
	tier2Missing := filterByImportance(missing, 2)
	tier3Missing := filterByImportance(missing, 3)

	if len(tier1Missing) > 0 {
		fmt.Println("\n🔴 TIER 1 - ESSENTIAL (Critical gaps!):")
		for _, v := range tier1Missing {
			fmt.Printf("   %-15s %s\n", v.VerseID, v.Reason)
		}
	}

	if len(tier2Missing) > 0 {
		fmt.Println("\n🟡 TIER 2 - IMPORTANT:")
		for _, v := range tier2Missing {
			fmt.Printf("   %-15s %s\n", v.VerseID, v.Reason)
		}
	}

	if len(tier3Missing) > 0 {
		fmt.Println("\n🟢 TIER 3 - SUPPORTING:")
		for _, v := range tier3Missing {
			fmt.Printf("   %-15s %s\n", v.VerseID, v.Reason)
		}
	}

	// Report what's already present
	fmt.Println("\n" + "-" + strings.Repeat("-", 79))
	fmt.Println("ALREADY PRESENT (good!)")
	fmt.Println("-" + strings.Repeat("-", 79))

	tier1Present := filterByImportance(present, 1)
	tier2Present := filterByImportance(present, 2)
	tier3Present := filterByImportance(present, 3)

	fmt.Printf("\n✅ Tier 1 present: %d/%d\n", len(tier1Present), len(filterByImportance(TrinityCanonicalVerses, 1)))
	for _, v := range tier1Present {
		fmt.Printf("   %-15s %s\n", v.VerseID, v.Reason)
	}

	fmt.Printf("\n✅ Tier 2 present: %d/%d\n", len(tier2Present), len(filterByImportance(TrinityCanonicalVerses, 2)))
	for _, v := range tier2Present {
		fmt.Printf("   %-15s %s\n", v.VerseID, v.Reason)
	}

	fmt.Printf("\n✅ Tier 3 present: %d/%d\n", len(tier3Present), len(filterByImportance(TrinityCanonicalVerses, 3)))
	for _, v := range tier3Present {
		fmt.Printf("   %-15s %s\n", v.VerseID, v.Reason)
	}

	// Generate INSERT statements for missing verses
	if len(missing) > 0 {
		fmt.Println("\n" + "=" + strings.Repeat("=", 79))
		fmt.Println("SQL TO ADD MISSING VERSES")
		fmt.Println("=" + strings.Repeat("=", 79))
		fmt.Println("\n-- Run this to add missing canonical verses to Trinity topic")
		fmt.Println("INSERT INTO api.topic_verses (topic_id, verse_id)")
		fmt.Println("SELECT 33509, v.id")
		fmt.Println("FROM api.verses v")
		fmt.Println("WHERE v.osis_verse_id IN (")

		for i, v := range missing {
			comma := ","
			if i == len(missing)-1 {
				comma = ""
			}
			fmt.Printf("    '%s'%s\n", v.VerseID, comma)
		}
		fmt.Println(")")
		fmt.Println("ON CONFLICT DO NOTHING;")
	}

	// Summary
	fmt.Println("\n" + "=" + strings.Repeat("=", 79))
	fmt.Println("SUMMARY")
	fmt.Println("=" + strings.Repeat("=", 79))
	fmt.Printf("Total canonical verses defined: %d\n", len(TrinityCanonicalVerses))
	fmt.Printf("Already in topic: %d (%.0f%%)\n", len(present), float64(len(present))/float64(len(TrinityCanonicalVerses))*100)
	fmt.Printf("Missing: %d\n", len(missing))
	fmt.Printf("  - Tier 1 (Essential): %d missing\n", len(tier1Missing))
	fmt.Printf("  - Tier 2 (Important): %d missing\n", len(tier2Missing))
	fmt.Printf("  - Tier 3 (Supporting): %d missing\n", len(tier3Missing))
}

func filterByImportance(verses []CanonicalVerse, importance int) []CanonicalVerse {
	var result []CanonicalVerse
	for _, v := range verses {
		if v.Importance == importance {
			result = append(result, v)
		}
	}
	return result
}
