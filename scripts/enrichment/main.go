package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"cloud.google.com/go/vertexai/genai"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"github.com/lib/pq"
)

// Verse represents a verse with its context
type Verse struct {
	VerseID       string   `db:"osis_verse_id" json:"osis_verse_id"`
	Book          string   `db:"book" json:"book"`
	Chapter       int      `db:"chapter" json:"chapter"`
	VerseNum      int      `db:"verse" json:"verse"`
	Text          string   `db:"text" json:"text"`
	CrossRefs     []string `json:"cross_refs,omitempty"`
	Topics        []string `json:"topics,omitempty"`
	ChapterText   string   `json:"chapter_context,omitempty"`
}

// EnrichmentResult holds both enrichment approaches for a verse
type EnrichmentResult struct {
	Verse            Verse    `json:"verse"`
	TheoAnnotations  []string `json:"theological_annotations"`
	SyntheticQueries []string `json:"synthetic_queries"`
	AugmentedText    string   `json:"augmented_text"`
}

// SampleConfig defines the sampling strategy
type SampleConfig struct {
	// Specific theologically significant verses to always include
	MustInclude []string
	// Number of random verses per testament
	RandomPerTestament int
	// Genres to sample from
	Genres []string
}

var defaultSampleConfig = SampleConfig{
	MustInclude: []string{
		// Trinity-related
		"Gen.1.26", "Gen.1.1", "Gen.11.7", "Isa.6.8",
		"Matt.28.19", "John.1.1", "John.1.14", "John.10.30",
		"2Cor.13.14", "1John.5.7",
		// Salvation
		"John.3.16", "Rom.3.23", "Rom.6.23", "Eph.2.8",
		"Acts.4.12", "Rom.10.9", "Titus.3.5",
		// Prophecy/Messianic
		"Isa.7.14", "Isa.53.5", "Mic.5.2", "Ps.22.16",
		"Dan.9.26", "Zech.12.10",
		// Wisdom/Practical
		"Prov.3.5", "Prov.3.6", "Ps.23.1", "Jer.29.11",
		// Difficult/Ambiguous
		"1Pet.3.19", "Heb.6.4", "Matt.16.18", "1Cor.15.29",
	},
	RandomPerTestament: 15,
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	godotenv.Load()

	ctx := context.Background()

	// Connect to PostgreSQL
	db, err := sqlx.Connect("postgres", os.Getenv("POSTGRES_URI"))
	if err != nil {
		return fmt.Errorf("connect to postgres: %w", err)
	}
	defer db.Close()

	// Initialize Vertex AI Gemini client (uses ADC)
	projectID := os.Getenv("GCP_PROJECT_ID")
	location := os.Getenv("GEMINI_LOCATION")
	if projectID == "" {
		projectID = "sola-scriptura-project-dev"
	}
	if location == "" {
		location = "global" // gemini-3 models require global location
	}
	client, err := genai.NewClient(ctx, projectID, location)
	if err != nil {
		return fmt.Errorf("create genai client: %w", err)
	}
	defer client.Close()

	// Get sample verses
	log.Println("Selecting sample verses...")
	verses, err := getSampleVerses(ctx, db, defaultSampleConfig)
	if err != nil {
		return fmt.Errorf("get sample verses: %w", err)
	}
	log.Printf("Selected %d verses for enrichment\n", len(verses))

	// Enrich each verse
	results := make([]EnrichmentResult, 0, len(verses))
	for i, verse := range verses {
		log.Printf("[%d/%d] Enriching %s...\n", i+1, len(verses), verse.VerseID)

		result, err := enrichVerse(ctx, client, verse)
		if err != nil {
			log.Printf("  Warning: failed to enrich %s: %v\n", verse.VerseID, err)
			continue
		}
		results = append(results, result)

		// Log a preview
		log.Printf("  Annotations: %v\n", result.TheoAnnotations[:min(3, len(result.TheoAnnotations))])
		log.Printf("  Queries: %v\n", result.SyntheticQueries[:min(2, len(result.SyntheticQueries))])
	}

	// Write results to JSON file
	outputFile := "enrichment_results.json"
	if err := writeResults(results, outputFile); err != nil {
		return fmt.Errorf("write results: %w", err)
	}
	log.Printf("Results written to %s\n", outputFile)

	// Also write a human-readable summary
	summaryFile := "enrichment_summary.md"
	if err := writeSummary(results, summaryFile); err != nil {
		return fmt.Errorf("write summary: %w", err)
	}
	log.Printf("Summary written to %s\n", summaryFile)

	return nil
}

func getSampleVerses(ctx context.Context, db *sqlx.DB, config SampleConfig) ([]Verse, error) {
	verses := make([]Verse, 0, len(config.MustInclude)+config.RandomPerTestament*2)

	// Get must-include verses
	if len(config.MustInclude) > 0 {
		query := `
			SELECT v.osis_verse_id, b.osis_id as book, v.chapter, v.verse, v.text
			FROM api.verses v
			JOIN api.books b ON v.book_id = b.id
			WHERE v.osis_verse_id = ANY($1)
		`
		var mustInclude []Verse
		if err := db.SelectContext(ctx, &mustInclude, query, pq.Array(config.MustInclude)); err != nil {
			return nil, fmt.Errorf("get must-include verses: %w", err)
		}
		verses = append(verses, mustInclude...)
	}

	// Get random OT verses
	queryOT := `
		SELECT v.osis_verse_id, b.osis_id as book, v.chapter, v.verse, v.text
		FROM api.verses v
		JOIN api.books b ON v.book_id = b.id
		WHERE b.testament = 'OT'
		AND v.osis_verse_id != ALL($1)
		ORDER BY RANDOM()
		LIMIT $2
	`
	var otVerses []Verse
	if err := db.SelectContext(ctx, &otVerses, queryOT, pq.Array(config.MustInclude), config.RandomPerTestament); err != nil {
		return nil, fmt.Errorf("get OT verses: %w", err)
	}
	verses = append(verses, otVerses...)

	// Get random NT verses
	queryNT := `
		SELECT v.osis_verse_id, b.osis_id as book, v.chapter, v.verse, v.text
		FROM api.verses v
		JOIN api.books b ON v.book_id = b.id
		WHERE b.testament = 'NT'
		AND v.osis_verse_id != ALL($1)
		ORDER BY RANDOM()
		LIMIT $2
	`
	var ntVerses []Verse
	if err := db.SelectContext(ctx, &ntVerses, queryNT, pq.Array(config.MustInclude), config.RandomPerTestament); err != nil {
		return nil, fmt.Errorf("get NT verses: %w", err)
	}
	verses = append(verses, ntVerses...)

	// Enrich with cross-references and topics
	for i := range verses {
		verses[i].CrossRefs = getCrossRefs(ctx, db, verses[i].VerseID)
		verses[i].Topics = getTopics(ctx, db, verses[i].VerseID)
		verses[i].ChapterText = getChapterContext(ctx, db, verses[i].Book, verses[i].Chapter, verses[i].VerseNum)
	}

	return verses, nil
}

func getCrossRefs(ctx context.Context, db *sqlx.DB, verseID string) []string {
	// Check if refs table has cross-reference data
	query := `
		SELECT v2.osis_verse_id
		FROM api.refs r
		JOIN api.verses v1 ON r.source_verse_id = v1.id
		JOIN api.verses v2 ON r.target_verse_id = v2.id
		WHERE v1.osis_verse_id = $1
		LIMIT 10
	`
	var refs []string
	db.SelectContext(ctx, &refs, query, verseID)
	return refs
}

func getTopics(ctx context.Context, db *sqlx.DB, verseID string) []string {
	query := `
		SELECT t.name
		FROM api.topics t
		JOIN api.topic_verses tv ON t.id = tv.topic_id
		JOIN api.verses v ON tv.verse_id = v.id
		WHERE v.osis_verse_id = $1
		LIMIT 10
	`
	var topics []string
	db.SelectContext(ctx, &topics, query, verseID)
	return topics
}

func getChapterContext(ctx context.Context, db *sqlx.DB, bookOsisID string, chapter, verse int) string {
	// Get surrounding verses (5 before, 5 after)
	query := `
		SELECT v.text
		FROM api.verses v
		JOIN api.books b ON v.book_id = b.id
		WHERE b.osis_id = $1 AND v.chapter = $2
		AND v.verse BETWEEN $3 AND $4
		ORDER BY v.verse
	`
	startVerse := max(1, verse-5)
	endVerse := verse + 5

	var texts []string
	db.SelectContext(ctx, &texts, query, bookOsisID, chapter, startVerse, endVerse)
	return strings.Join(texts, " ")
}

func enrichVerse(ctx context.Context, client *genai.Client, verse Verse) (EnrichmentResult, error) {
	result := EnrichmentResult{Verse: verse}

	// Build context for the LLM
	contextInfo := buildContextInfo(verse)

	// Generate theological annotations
	annotations, err := generateAnnotations(ctx, client, verse, contextInfo)
	if err != nil {
		return result, fmt.Errorf("generate annotations: %w", err)
	}
	result.TheoAnnotations = annotations

	// Generate synthetic queries
	queries, err := generateSyntheticQueries(ctx, client, verse, contextInfo)
	if err != nil {
		return result, fmt.Errorf("generate queries: %w", err)
	}
	result.SyntheticQueries = queries

	// Build augmented text for Option 1
	result.AugmentedText = fmt.Sprintf("%s [Themes: %s]", verse.Text, strings.Join(annotations, ", "))

	return result, nil
}

func buildContextInfo(verse Verse) string {
	var parts []string

	if len(verse.CrossRefs) > 0 {
		parts = append(parts, fmt.Sprintf("Cross-references: %s", strings.Join(verse.CrossRefs, ", ")))
	}
	if len(verse.Topics) > 0 {
		parts = append(parts, fmt.Sprintf("Associated topics: %s", strings.Join(verse.Topics, ", ")))
	}
	if verse.ChapterText != "" {
		parts = append(parts, fmt.Sprintf("Surrounding context: %s", verse.ChapterText))
	}

	return strings.Join(parts, "\n")
}

func generateAnnotations(ctx context.Context, client *genai.Client, verse Verse, contextInfo string) ([]string, error) {
	prompt := fmt.Sprintf(`You are a biblical scholar with expertise in systematic theology, biblical languages, and hermeneutics.

Analyze this Bible verse and provide 5-8 theological themes, concepts, or doctrines that this verse relates to or supports.

VERSE: %s %d:%d
TEXT: "%s"

CONTEXT:
%s

INSTRUCTIONS:
- Include both explicit themes (directly stated) and implicit themes (theologically derived)
- Use standard theological terminology (e.g., "Trinity", "Justification", "Sanctification", "Eschatology")
- Include relevant Hebrew/Greek concepts if applicable (e.g., "hesed/covenant love", "logos/divine word")
- Consider how this verse is used in systematic theology and doctrinal discussions
- Think about what topics a Bible student might be searching for when they need this verse

Return ONLY a JSON array of strings, no explanation. Example:
["Theme 1", "Theme 2", "Theme 3"]`,
		verse.Book, verse.Chapter, verse.VerseNum, verse.Text, contextInfo)

	model := client.GenerativeModel("gemini-3-flash-preview")
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, err
	}

	// Parse response
	text := extractText(resp)
	return parseJSONArray(text)
}

func generateSyntheticQueries(ctx context.Context, client *genai.Client, verse Verse, contextInfo string) ([]string, error) {
	prompt := fmt.Sprintf(`You are helping build a Bible search engine. For the given verse, generate 5-7 natural language search queries that a user might type when looking for this verse.

VERSE: %s %d:%d
TEXT: "%s"

CONTEXT:
%s

INSTRUCTIONS:
- Write queries as a real user would search (natural language, not keywords)
- Include both specific queries ("What does the Bible say about X?") and exploratory queries ("verses about Y")
- Include queries for both obvious themes AND subtle/implicit themes
- Consider theological questions this verse answers
- Consider practical life questions this verse addresses
- Vary query styles: questions, topic searches, doctrinal lookups

Return ONLY a JSON array of strings, no explanation. Example:
["What does the Bible say about X?", "verses about Y", "biblical teaching on Z"]`,
		verse.Book, verse.Chapter, verse.VerseNum, verse.Text, contextInfo)

	model := client.GenerativeModel("gemini-3-flash-preview")
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, err
	}

	text := extractText(resp)
	return parseJSONArray(text)
}

func extractText(resp *genai.GenerateContentResponse) string {
	if resp == nil || len(resp.Candidates) == 0 {
		return ""
	}
	candidate := resp.Candidates[0]
	if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
		return ""
	}

	var text string
	for _, part := range candidate.Content.Parts {
		if t, ok := part.(genai.Text); ok {
			text += string(t)
		}
	}
	return text
}

func parseJSONArray(text string) ([]string, error) {
	// Clean up the response - remove markdown code blocks if present
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	var result []string
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("parse JSON array: %w (raw: %s)", err, text)
	}
	return result, nil
}

func writeResults(results []EnrichmentResult, filename string) error {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func writeSummary(results []EnrichmentResult, filename string) error {
	var sb strings.Builder

	sb.WriteString("# Semantic Enrichment Prototype Results\n\n")
	sb.WriteString(fmt.Sprintf("**Total verses processed:** %d\n\n", len(results)))
	sb.WriteString("---\n\n")

	for _, r := range results {
		sb.WriteString(fmt.Sprintf("## %s\n\n", r.Verse.VerseID))
		sb.WriteString(fmt.Sprintf("**Text:** %s\n\n", r.Verse.Text))

		sb.WriteString("**Theological Annotations:**\n")
		for _, a := range r.TheoAnnotations {
			sb.WriteString(fmt.Sprintf("- %s\n", a))
		}
		sb.WriteString("\n")

		sb.WriteString("**Synthetic Queries:**\n")
		for _, q := range r.SyntheticQueries {
			sb.WriteString(fmt.Sprintf("- %s\n", q))
		}
		sb.WriteString("\n")

		sb.WriteString(fmt.Sprintf("**Augmented Text:** %s\n\n", r.AugmentedText))
		sb.WriteString("---\n\n")
	}

	return os.WriteFile(filename, []byte(sb.String()), 0644)
}
