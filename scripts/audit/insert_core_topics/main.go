package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"github.com/lib/pq"
)

// CanonicalVerse represents a verse for a topic
type CanonicalVerse struct {
	VerseID    string
	Importance int // 1 = essential, 2 = important, 3 = supporting
}

// TopicDefinition defines a topic with its canonical verses
type TopicDefinition struct {
	Name        string
	Slug        string
	Category    string
	Description string
	Verses      []CanonicalVerse
}

// CoreTopics contains Claude's curated canonical verses for core theological concepts
var CoreTopics = []TopicDefinition{
	{
		Name:     "Salvation",
		Slug:     "salvation",
		Category: "concept",
		Description: "The deliverance from sin and its consequences through faith in Jesus Christ. " +
			"Includes justification, regeneration, and the gift of eternal life.",
		Verses: []CanonicalVerse{
			// Tier 1: Essential
			{VerseID: "Eph.2.8", Importance: 1},  // By grace through faith
			{VerseID: "Eph.2.9", Importance: 1},  // Not of works
			{VerseID: "Rom.10.9", Importance: 1}, // Confess and believe
			{VerseID: "Rom.10.10", Importance: 1},
			{VerseID: "John.3.16", Importance: 1},  // God so loved
			{VerseID: "John.3.17", Importance: 1},  // Not to condemn
			{VerseID: "Acts.4.12", Importance: 1},  // No other name
			{VerseID: "Titus.3.5", Importance: 1},  // Not by works of righteousness
			{VerseID: "Rom.6.23", Importance: 1},   // Wages of sin / gift of God
			{VerseID: "John.14.6", Importance: 1},  // I am the way
			// Tier 2: Important
			{VerseID: "Rom.3.23", Importance: 2},  // All have sinned
			{VerseID: "Rom.3.24", Importance: 2},  // Justified freely
			{VerseID: "Rom.5.8", Importance: 2},   // While we were sinners
			{VerseID: "Rom.5.9", Importance: 2},   // Justified by his blood
			{VerseID: "Acts.16.30", Importance: 2}, // What must I do
			{VerseID: "Acts.16.31", Importance: 2}, // Believe on the Lord
			{VerseID: "John.1.12", Importance: 2},  // Right to become children
			{VerseID: "John.5.24", Importance: 2},  // Passed from death to life
			{VerseID: "2Cor.5.17", Importance: 2},  // New creation
			{VerseID: "1Pet.1.3", Importance: 2},   // Born again to living hope
			// Tier 3: Supporting
			{VerseID: "Isa.53.5", Importance: 3},   // By his stripes healed
			{VerseID: "Isa.53.6", Importance: 3},   // All we like sheep
			{VerseID: "Rom.8.1", Importance: 3},    // No condemnation
			{VerseID: "Gal.2.16", Importance: 3},   // Not justified by works of law
			{VerseID: "Phil.2.12", Importance: 3},  // Work out your salvation
			{VerseID: "Heb.7.25", Importance: 3},   // Able to save completely
			{VerseID: "1John.5.11", Importance: 3}, // God gave us eternal life
			{VerseID: "1John.5.12", Importance: 3}, // He who has the Son
			{VerseID: "Rev.3.20", Importance: 3},   // Behold I stand at the door
			{VerseID: "John.10.28", Importance: 3}, // I give them eternal life
		},
	},
	{
		Name:     "Grace",
		Slug:     "grace",
		Category: "concept",
		Description: "God's unmerited favor toward sinners. The foundation of salvation, " +
			"freely given through Christ, not earned by human effort.",
		Verses: []CanonicalVerse{
			// Tier 1: Essential
			{VerseID: "Eph.2.8", Importance: 1},   // By grace through faith
			{VerseID: "Eph.2.9", Importance: 1},   // Not of works
			{VerseID: "Rom.5.8", Importance: 1},   // While we were sinners
			{VerseID: "2Cor.12.9", Importance: 1}, // My grace is sufficient
			{VerseID: "John.1.14", Importance: 1}, // Full of grace and truth
			{VerseID: "John.1.16", Importance: 1}, // Grace upon grace
			{VerseID: "Titus.2.11", Importance: 1}, // Grace has appeared
			{VerseID: "Rom.3.24", Importance: 1},   // Justified freely by grace
			{VerseID: "Rom.5.20", Importance: 1},   // Where sin increased
			{VerseID: "Rom.5.21", Importance: 1},   // Grace might reign
			// Tier 2: Important
			{VerseID: "Titus.3.7", Importance: 2},  // Justified by his grace
			{VerseID: "Rom.11.6", Importance: 2},   // If by grace, not works
			{VerseID: "Gal.2.21", Importance: 2},   // Do not nullify grace
			{VerseID: "Heb.4.16", Importance: 2},   // Throne of grace
			{VerseID: "1Pet.5.10", Importance: 2},  // God of all grace
			{VerseID: "2Cor.8.9", Importance: 2},   // Though rich became poor
			{VerseID: "Acts.15.11", Importance: 2}, // Through grace we are saved
			{VerseID: "Rom.6.14", Importance: 2},   // Not under law but grace
			{VerseID: "Eph.1.7", Importance: 2},    // Riches of his grace
			{VerseID: "2Tim.1.9", Importance: 2},   // Called according to grace
			// Tier 3: Supporting
			{VerseID: "James.4.6", Importance: 3},  // Gives grace to humble
			{VerseID: "1Pet.4.10", Importance: 3},  // Stewards of grace
			{VerseID: "Rom.12.6", Importance: 3},   // Gifts differing by grace
			{VerseID: "1Cor.15.10", Importance: 3}, // By grace I am what I am
			{VerseID: "Gal.1.15", Importance: 3},   // Called by his grace
			{VerseID: "Col.4.6", Importance: 3},    // Speech seasoned with grace
			{VerseID: "Heb.12.15", Importance: 3},  // Fall short of grace
			{VerseID: "2Pet.3.18", Importance: 3},  // Grow in grace
		},
	},
	{
		Name:     "Faith",
		Slug:     "faith",
		Category: "concept",
		Description: "Trust and confidence in God and His promises. The means by which salvation " +
			"is received, and the foundation of the Christian life.",
		Verses: []CanonicalVerse{
			// Tier 1: Essential
			{VerseID: "Heb.11.1", Importance: 1},  // Substance of things hoped for
			{VerseID: "Heb.11.6", Importance: 1},  // Without faith impossible to please
			{VerseID: "Rom.10.17", Importance: 1}, // Faith comes by hearing
			{VerseID: "Eph.2.8", Importance: 1},   // By grace through faith
			{VerseID: "Rom.1.17", Importance: 1},  // The just shall live by faith
			{VerseID: "Gal.2.20", Importance: 1},  // I live by faith
			{VerseID: "Rom.5.1", Importance: 1},   // Justified by faith
			{VerseID: "James.2.17", Importance: 1}, // Faith without works is dead
			{VerseID: "James.2.26", Importance: 1}, // Faith without works is dead
			{VerseID: "2Cor.5.7", Importance: 1},   // Walk by faith not sight
			// Tier 2: Important
			{VerseID: "Mark.11.22", Importance: 2},  // Have faith in God
			{VerseID: "Mark.11.23", Importance: 2},  // Say to this mountain
			{VerseID: "Mark.11.24", Importance: 2},  // Whatever you ask in prayer
			{VerseID: "Matt.17.20", Importance: 2},  // Faith as a mustard seed
			{VerseID: "Rom.4.5", Importance: 2},     // Faith counted as righteousness
			{VerseID: "Rom.4.20", Importance: 2},    // Abraham did not waver
			{VerseID: "Rom.4.21", Importance: 2},    // Fully persuaded
			{VerseID: "Gal.3.11", Importance: 2},    // Just shall live by faith
			{VerseID: "Heb.11.3", Importance: 2},    // By faith we understand
			{VerseID: "1Pet.1.7", Importance: 2},    // Trial of your faith
			// Tier 3: Supporting
			{VerseID: "Heb.11.7", Importance: 3},   // By faith Noah
			{VerseID: "Heb.11.8", Importance: 3},   // By faith Abraham
			{VerseID: "Heb.11.11", Importance: 3},  // By faith Sarah
			{VerseID: "Heb.11.17", Importance: 3},  // By faith Abraham offered Isaac
			{VerseID: "Heb.11.24", Importance: 3},  // By faith Moses
			{VerseID: "Heb.12.2", Importance: 3},   // Author and finisher of faith
			{VerseID: "1John.5.4", Importance: 3},  // Overcomes the world - faith
			{VerseID: "Gal.5.6", Importance: 3},    // Faith working through love
			{VerseID: "Col.2.6", Importance: 3},    // Walk in him
			{VerseID: "1Tim.6.12", Importance: 3},  // Fight the good fight of faith
		},
	},
	{
		Name:     "Atonement",
		Slug:     "atonement",
		Category: "concept",
		Description: "The reconciliation of God and humanity through Christ's sacrificial death. " +
			"Christ bore the penalty for sin, satisfying divine justice.",
		Verses: []CanonicalVerse{
			// Tier 1: Essential
			{VerseID: "Rom.3.25", Importance: 1},   // Propitiation through faith
			{VerseID: "1John.2.2", Importance: 1},  // Propitiation for our sins
			{VerseID: "1John.4.10", Importance: 1}, // Sent his Son as propitiation
			{VerseID: "Isa.53.5", Importance: 1},   // Wounded for our transgressions
			{VerseID: "Isa.53.6", Importance: 1},   // Lord laid on him our iniquity
			{VerseID: "Heb.9.22", Importance: 1},   // Without shedding of blood
			{VerseID: "1Pet.2.24", Importance: 1},  // Bore our sins in his body
			{VerseID: "2Cor.5.21", Importance: 1},  // Made him to be sin for us
			{VerseID: "Rom.5.10", Importance: 1},   // Reconciled by his death
			{VerseID: "Rom.5.11", Importance: 1},   // Received the atonement
			// Tier 2: Important
			{VerseID: "Lev.17.11", Importance: 2},  // Life of the flesh is in blood
			{VerseID: "Heb.9.12", Importance: 2},   // By his own blood
			{VerseID: "Heb.9.14", Importance: 2},   // Blood of Christ cleanse
			{VerseID: "Col.1.20", Importance: 2},   // Peace through blood of cross
			{VerseID: "Eph.1.7", Importance: 2},    // Redemption through his blood
			{VerseID: "1Pet.1.18", Importance: 2},  // Not redeemed with silver
			{VerseID: "1Pet.1.19", Importance: 2},  // But with precious blood
			{VerseID: "Rev.5.9", Importance: 2},    // Redeemed us by your blood
			{VerseID: "Heb.10.10", Importance: 2},  // Sanctified through offering
			{VerseID: "Heb.10.12", Importance: 2},  // One sacrifice for sins forever
			// Tier 3: Supporting
			{VerseID: "Lev.16.15", Importance: 3},  // Day of Atonement sacrifice
			{VerseID: "Lev.16.16", Importance: 3},  // Make atonement for holy place
			{VerseID: "Isa.53.10", Importance: 3},  // Offering for guilt
			{VerseID: "Isa.53.11", Importance: 3},  // Justify many
			{VerseID: "Isa.53.12", Importance: 3},  // Bore the sin of many
			{VerseID: "Gal.3.13", Importance: 3},   // Redeemed us from curse
			{VerseID: "John.1.29", Importance: 3},  // Lamb of God takes away sin
			{VerseID: "Heb.2.17", Importance: 3},   // Make propitiation
		},
	},
	{
		Name:     "Redemption",
		Slug:     "redemption",
		Category: "concept",
		Description: "The act of buying back or setting free from bondage. Christ's payment of the " +
			"price to free humanity from slavery to sin.",
		Verses: []CanonicalVerse{
			// Tier 1: Essential
			{VerseID: "Eph.1.7", Importance: 1},    // Redemption through his blood
			{VerseID: "Col.1.14", Importance: 1},   // Redemption, forgiveness of sins
			{VerseID: "Rom.3.24", Importance: 1},   // Redemption in Christ Jesus
			{VerseID: "Titus.2.14", Importance: 1}, // Redeem us from all iniquity
			{VerseID: "1Pet.1.18", Importance: 1},  // Not redeemed with silver/gold
			{VerseID: "1Pet.1.19", Importance: 1},  // Precious blood of Christ
			{VerseID: "Gal.3.13", Importance: 1},   // Redeemed us from curse
			{VerseID: "Gal.4.5", Importance: 1},    // Redeem those under law
			{VerseID: "Rev.5.9", Importance: 1},    // Redeemed us by your blood
			{VerseID: "Heb.9.12", Importance: 1},   // Obtained eternal redemption
			// Tier 2: Important
			{VerseID: "Isa.44.22", Importance: 2},  // I have redeemed you
			{VerseID: "Isa.52.3", Importance: 2},   // Redeemed without money
			{VerseID: "Isa.63.9", Importance: 2},   // In his love he redeemed them
			{VerseID: "Ps.130.7", Importance: 2},   // With him is plentiful redemption
			{VerseID: "Ps.111.9", Importance: 2},   // He sent redemption
			{VerseID: "Luke.1.68", Importance: 2},  // Visited and redeemed his people
			{VerseID: "Luke.21.28", Importance: 2}, // Redemption draws near
			{VerseID: "Rom.8.23", Importance: 2},   // Redemption of our bodies
			{VerseID: "Eph.4.30", Importance: 2},   // Sealed for day of redemption
			{VerseID: "1Cor.1.30", Importance: 2},  // Christ our redemption
			// Tier 3: Supporting
			{VerseID: "Exod.6.6", Importance: 3},   // I will redeem you
			{VerseID: "Exod.15.13", Importance: 3}, // People you have redeemed
			{VerseID: "Ruth.4.4", Importance: 3},   // Kinsman redeemer concept
			{VerseID: "Job.19.25", Importance: 3},  // I know my redeemer lives
			{VerseID: "Ps.19.14", Importance: 3},   // O Lord my rock and redeemer
			{VerseID: "Ps.49.15", Importance: 3},   // God will redeem my soul
			{VerseID: "Isa.43.1", Importance: 3},   // I have redeemed you
			{VerseID: "Mark.10.45", Importance: 3}, // Give his life as ransom
		},
	},
	{
		Name:     "Justification",
		Slug:     "justification",
		Category: "concept",
		Description: "The judicial act of God declaring sinners righteous through faith in Christ. " +
			"A legal declaration, not a moral transformation.",
		Verses: []CanonicalVerse{
			// Tier 1: Essential
			{VerseID: "Rom.3.24", Importance: 1},  // Justified freely by grace
			{VerseID: "Rom.3.28", Importance: 1},  // Justified by faith apart from works
			{VerseID: "Rom.4.5", Importance: 1},   // Faith counted as righteousness
			{VerseID: "Rom.5.1", Importance: 1},   // Justified by faith, peace with God
			{VerseID: "Rom.5.9", Importance: 1},   // Justified by his blood
			{VerseID: "Gal.2.16", Importance: 1},  // Not justified by works of law
			{VerseID: "Gal.3.11", Importance: 1},  // No one justified by law
			{VerseID: "Rom.8.30", Importance: 1},  // Those he called he justified
			{VerseID: "Rom.8.33", Importance: 1},  // God who justifies
			{VerseID: "Titus.3.7", Importance: 1}, // Justified by his grace
			// Tier 2: Important
			{VerseID: "Rom.4.2", Importance: 2},   // Abraham not justified by works
			{VerseID: "Rom.4.3", Importance: 2},   // Believed God, credited righteousness
			{VerseID: "Rom.4.25", Importance: 2},  // Raised for our justification
			{VerseID: "Rom.5.16", Importance: 2},  // Justification from many trespasses
			{VerseID: "Rom.5.18", Importance: 2},  // One act of righteousness
			{VerseID: "Gal.3.24", Importance: 2},  // Law was tutor to bring us to Christ
			{VerseID: "Phil.3.9", Importance: 2},  // Righteousness from God by faith
			{VerseID: "Acts.13.38", Importance: 2}, // Forgiveness proclaimed
			{VerseID: "Acts.13.39", Importance: 2}, // Justified from all things
			{VerseID: "2Cor.5.21", Importance: 2},  // Become righteousness of God
			// Tier 3: Supporting
			{VerseID: "James.2.21", Importance: 3}, // Abraham justified by works
			{VerseID: "James.2.24", Importance: 3}, // Justified by works not faith alone
			{VerseID: "James.2.25", Importance: 3}, // Rahab justified by works
			{VerseID: "Rom.3.20", Importance: 3},   // No flesh justified by law
			{VerseID: "Rom.3.26", Importance: 3},   // Just and justifier
			{VerseID: "Isa.53.11", Importance: 3},  // Servant will justify many
			{VerseID: "Luke.18.14", Importance: 3}, // Went home justified
			{VerseID: "1Cor.6.11", Importance: 3},  // You were justified
		},
	},
	{
		Name:     "Sanctification",
		Slug:     "sanctification",
		Category: "concept",
		Description: "The process of being made holy. Both a positional reality in Christ and " +
			"a progressive work of the Spirit conforming believers to Christ's image.",
		Verses: []CanonicalVerse{
			// Tier 1: Essential
			{VerseID: "1Thess.4.3", Importance: 1},  // This is God's will: sanctification
			{VerseID: "1Thess.5.23", Importance: 1}, // God of peace sanctify you
			{VerseID: "Heb.10.10", Importance: 1},   // Sanctified through the body of Christ
			{VerseID: "Heb.10.14", Importance: 1},   // Perfected those being sanctified
			{VerseID: "1Cor.1.30", Importance: 1},   // Christ our sanctification
			{VerseID: "1Cor.6.11", Importance: 1},   // You were sanctified
			{VerseID: "2Thess.2.13", Importance: 1}, // Sanctification by the Spirit
			{VerseID: "1Pet.1.2", Importance: 1},    // Sanctification of the Spirit
			{VerseID: "John.17.17", Importance: 1},  // Sanctify them in truth
			{VerseID: "Rom.6.22", Importance: 1},    // Fruit leading to sanctification
			// Tier 2: Important
			{VerseID: "Heb.12.14", Importance: 2},  // Without holiness no one will see Lord
			{VerseID: "Rom.12.1", Importance: 2},   // Present bodies as living sacrifice
			{VerseID: "Rom.12.2", Importance: 2},   // Be transformed by renewing
			{VerseID: "2Cor.7.1", Importance: 2},   // Perfecting holiness
			{VerseID: "Phil.2.12", Importance: 2},  // Work out your salvation
			{VerseID: "Phil.2.13", Importance: 2},  // God works in you
			{VerseID: "Gal.5.16", Importance: 2},   // Walk by the Spirit
			{VerseID: "Col.3.5", Importance: 2},    // Put to death what is earthly
			{VerseID: "Eph.5.26", Importance: 2},   // Sanctify her by washing
			{VerseID: "1John.3.3", Importance: 2},  // Purifies himself
			// Tier 3: Supporting
			{VerseID: "Lev.20.7", Importance: 3},   // Consecrate yourselves, be holy
			{VerseID: "1Pet.1.15", Importance: 3},  // Be holy as I am holy
			{VerseID: "1Pet.1.16", Importance: 3},  // Be holy for I am holy
			{VerseID: "Rom.8.29", Importance: 3},   // Conformed to image of Son
			{VerseID: "2Cor.3.18", Importance: 3},  // Transformed into his image
			{VerseID: "Eph.4.24", Importance: 3},   // Put on new self
			{VerseID: "Col.3.10", Importance: 3},   // Put on new self
			{VerseID: "Heb.13.12", Importance: 3},  // Jesus sanctify through blood
		},
	},
	{
		Name:     "Resurrection",
		Slug:     "resurrection",
		Category: "concept",
		Description: "The bodily rising from the dead. Christ's resurrection is the foundation of " +
			"Christian faith and guarantees the future resurrection of believers.",
		Verses: []CanonicalVerse{
			// Tier 1: Essential
			{VerseID: "1Cor.15.3", Importance: 1},  // Christ died for our sins
			{VerseID: "1Cor.15.4", Importance: 1},  // He was raised on third day
			{VerseID: "1Cor.15.14", Importance: 1}, // If Christ not raised, faith vain
			{VerseID: "1Cor.15.17", Importance: 1}, // If Christ not raised, still in sins
			{VerseID: "1Cor.15.20", Importance: 1}, // Christ has been raised, firstfruits
			{VerseID: "Rom.6.9", Importance: 1},    // Christ raised, will not die again
			{VerseID: "John.11.25", Importance: 1}, // I am the resurrection and life
			{VerseID: "John.11.26", Importance: 1}, // Whoever believes will never die
			{VerseID: "Rom.8.11", Importance: 1},   // Spirit will give life to bodies
			{VerseID: "1Pet.1.3", Importance: 1},   // Living hope through resurrection
			// Tier 2: Important
			{VerseID: "Phil.3.10", Importance: 2},   // Know power of his resurrection
			{VerseID: "Phil.3.11", Importance: 2},   // Attain to the resurrection
			{VerseID: "1Cor.15.42", Importance: 2},  // Raised imperishable
			{VerseID: "1Cor.15.43", Importance: 2},  // Raised in glory
			{VerseID: "1Cor.15.44", Importance: 2},  // Raised a spiritual body
			{VerseID: "1Cor.15.52", Importance: 2},  // Dead will be raised imperishable
			{VerseID: "1Cor.15.54", Importance: 2},  // Death swallowed up in victory
			{VerseID: "1Cor.15.55", Importance: 2},  // Where O death is your victory
			{VerseID: "John.5.28", Importance: 2},   // Hour coming when dead will hear
			{VerseID: "John.5.29", Importance: 2},   // Resurrection of life/judgment
			// Tier 3: Supporting
			{VerseID: "Matt.28.6", Importance: 3},  // He is not here, he has risen
			{VerseID: "Luke.24.6", Importance: 3},  // He is not here, he has risen
			{VerseID: "Acts.2.24", Importance: 3},  // God raised him up
			{VerseID: "Acts.2.32", Importance: 3},  // God has raised this Jesus
			{VerseID: "Acts.17.31", Importance: 3}, // Raised him from the dead
			{VerseID: "Rom.4.25", Importance: 3},   // Raised for our justification
			{VerseID: "1Thess.4.14", Importance: 3}, // God will bring with him
			{VerseID: "1Thess.4.16", Importance: 3}, // Dead in Christ will rise first
			{VerseID: "Rev.20.6", Importance: 3},    // First resurrection
			{VerseID: "Dan.12.2", Importance: 3},    // Many who sleep shall awake
		},
	},
	{
		Name:     "Holy Spirit",
		Slug:     "holy-spirit",
		Category: "concept",
		Description: "The third person of the Trinity. The Spirit convicts, regenerates, indwells, " +
			"seals, and empowers believers, producing fruit and distributing spiritual gifts.",
		Verses: []CanonicalVerse{
			// Tier 1: Essential
			{VerseID: "John.14.16", Importance: 1},  // Another Helper
			{VerseID: "John.14.17", Importance: 1},  // Spirit of truth
			{VerseID: "John.14.26", Importance: 1},  // Will teach you all things
			{VerseID: "John.16.7", Importance: 1},   // I will send the Helper
			{VerseID: "John.16.8", Importance: 1},   // Convict world of sin
			{VerseID: "John.16.13", Importance: 1},  // Guide you into all truth
			{VerseID: "Acts.1.8", Importance: 1},    // Receive power when Spirit comes
			{VerseID: "Acts.2.4", Importance: 1},    // Filled with Holy Spirit
			{VerseID: "Rom.8.9", Importance: 1},     // Spirit of God dwells in you
			{VerseID: "1Cor.6.19", Importance: 1},   // Body is temple of Spirit
			// Tier 2: Important
			{VerseID: "Gal.5.22", Importance: 2},   // Fruit of the Spirit
			{VerseID: "Gal.5.23", Importance: 2},   // Fruit continued
			{VerseID: "Eph.1.13", Importance: 2},   // Sealed with Spirit
			{VerseID: "Eph.1.14", Importance: 2},   // Guarantee of inheritance
			{VerseID: "Eph.5.18", Importance: 2},   // Be filled with Spirit
			{VerseID: "Rom.8.14", Importance: 2},   // Led by Spirit are sons
			{VerseID: "Rom.8.16", Importance: 2},   // Spirit testifies we are children
			{VerseID: "Rom.8.26", Importance: 2},   // Spirit helps our weakness
			{VerseID: "1Cor.12.4", Importance: 2},  // Varieties of gifts, same Spirit
			{VerseID: "1Cor.12.11", Importance: 2}, // Same Spirit apportions to each
			// Tier 3: Supporting
			{VerseID: "Titus.3.5", Importance: 3},   // Renewal by Holy Spirit
			{VerseID: "2Cor.3.17", Importance: 3},   // Where Spirit is, freedom
			{VerseID: "Gal.5.16", Importance: 3},    // Walk by Spirit
			{VerseID: "Gal.5.17", Importance: 3},    // Flesh against Spirit
			{VerseID: "Gal.5.25", Importance: 3},    // If we live by Spirit, walk by Spirit
			{VerseID: "1John.4.13", Importance: 3},  // He has given us of his Spirit
			{VerseID: "Rom.5.5", Importance: 3},     // Love poured through Spirit
			{VerseID: "2Cor.1.22", Importance: 3},   // Given Spirit as guarantee
			{VerseID: "Joel.2.28", Importance: 3},   // I will pour out my Spirit
			{VerseID: "Acts.2.17", Importance: 3},   // I will pour out my Spirit
		},
	},
	{
		Name:     "Second Coming",
		Slug:     "second-coming",
		Category: "concept",
		Description: "The future, visible, bodily return of Jesus Christ. He will come to judge " +
			"the living and the dead and establish His kingdom.",
		Verses: []CanonicalVerse{
			// Tier 1: Essential
			{VerseID: "Acts.1.11", Importance: 1},    // Will come in same way
			{VerseID: "Matt.24.30", Importance: 1},   // See Son of Man coming
			{VerseID: "Matt.24.44", Importance: 1},   // Be ready, coming at unexpected hour
			{VerseID: "1Thess.4.16", Importance: 1},  // Lord will descend from heaven
			{VerseID: "1Thess.4.17", Importance: 1},  // Caught up together
			{VerseID: "Titus.2.13", Importance: 1},   // Blessed hope, glorious appearing
			{VerseID: "Rev.1.7", Importance: 1},      // He is coming with clouds
			{VerseID: "John.14.3", Importance: 1},    // I will come again
			{VerseID: "Heb.9.28", Importance: 1},     // Will appear a second time
			{VerseID: "Rev.22.12", Importance: 1},    // Behold I am coming soon
			// Tier 2: Important
			{VerseID: "Matt.25.31", Importance: 2},   // Son of Man comes in glory
			{VerseID: "Matt.24.27", Importance: 2},   // Coming like lightning
			{VerseID: "Matt.24.36", Importance: 2},   // No one knows the day or hour
			{VerseID: "2Pet.3.10", Importance: 2},    // Day of Lord like a thief
			{VerseID: "2Pet.3.13", Importance: 2},    // New heavens and new earth
			{VerseID: "1Cor.15.23", Importance: 2},   // Those who belong to Christ at his coming
			{VerseID: "Phil.3.20", Importance: 2},    // We await a Savior from heaven
			{VerseID: "1John.3.2", Importance: 2},    // When he appears we shall be like him
			{VerseID: "James.5.8", Importance: 2},    // Coming of Lord is at hand
			{VerseID: "Rev.19.11", Importance: 2},    // Heaven opened, white horse
			// Tier 3: Supporting
			{VerseID: "Matt.25.13", Importance: 3},   // Watch, you do not know
			{VerseID: "Mark.13.26", Importance: 3},   // Son of Man coming in clouds
			{VerseID: "Luke.21.27", Importance: 3},   // Son of Man coming in cloud
			{VerseID: "1Thess.5.2", Importance: 3},   // Day of Lord like a thief
			{VerseID: "2Thess.1.7", Importance: 3},   // Lord Jesus revealed from heaven
			{VerseID: "2Thess.2.1", Importance: 3},   // Our gathering together to him
			{VerseID: "Rev.22.20", Importance: 3},    // Surely I am coming soon
			{VerseID: "Zech.14.4", Importance: 3},    // Feet will stand on Mount of Olives
		},
	},
	{
		Name:     "Heaven",
		Slug:     "heaven",
		Category: "concept",
		Description: "The dwelling place of God and the eternal home of believers. " +
			"A place of perfect joy, worship, and communion with God.",
		Verses: []CanonicalVerse{
			// Tier 1: Essential
			{VerseID: "John.14.2", Importance: 1},   // In my Father's house many rooms
			{VerseID: "John.14.3", Importance: 1},   // I go to prepare a place
			{VerseID: "Phil.3.20", Importance: 1},   // Our citizenship is in heaven
			{VerseID: "Rev.21.1", Importance: 1},    // New heaven and new earth
			{VerseID: "Rev.21.4", Importance: 1},    // Wipe away every tear
			{VerseID: "Rev.21.3", Importance: 1},    // God will dwell with them
			{VerseID: "2Cor.5.1", Importance: 1},    // House not made with hands
			{VerseID: "1Pet.1.4", Importance: 1},    // Inheritance imperishable
			{VerseID: "Matt.6.20", Importance: 1},   // Store up treasures in heaven
			{VerseID: "Heb.11.16", Importance: 1},   // Better country, heavenly one
			// Tier 2: Important
			{VerseID: "Rev.22.3", Importance: 2},    // No more curse
			{VerseID: "Rev.22.4", Importance: 2},    // See his face
			{VerseID: "Rev.22.5", Importance: 2},    // No more night
			{VerseID: "Rev.21.21", Importance: 2},   // Street of pure gold
			{VerseID: "Rev.21.23", Importance: 2},   // Glory of God gives light
			{VerseID: "1Cor.2.9", Importance: 2},    // Eye has not seen
			{VerseID: "Col.1.5", Importance: 2},     // Hope laid up in heaven
			{VerseID: "2Cor.12.2", Importance: 2},   // Caught up to third heaven
			{VerseID: "Heb.12.22", Importance: 2},   // Heavenly Jerusalem
			{VerseID: "Matt.5.12", Importance: 2},   // Great is your reward in heaven
			// Tier 3: Supporting
			{VerseID: "Luke.23.43", Importance: 3},  // Today with me in paradise
			{VerseID: "2Pet.3.13", Importance: 3},   // New heavens, righteousness dwells
			{VerseID: "Rev.7.17", Importance: 3},    // Lamb will be their shepherd
			{VerseID: "Isa.65.17", Importance: 3},   // I create new heavens
			{VerseID: "Ps.16.11", Importance: 3},    // Fullness of joy, pleasures forevermore
			{VerseID: "Matt.22.30", Importance: 3},  // Like angels in heaven
			{VerseID: "1Thess.4.17", Importance: 3}, // Always be with the Lord
			{VerseID: "Rev.4.11", Importance: 3},    // Worthy are you, Lord
		},
	},
	{
		Name:     "Hell",
		Slug:     "hell",
		Category: "concept",
		Description: "The place of eternal punishment for the wicked. Characterized by separation " +
			"from God, conscious torment, and everlasting duration.",
		Verses: []CanonicalVerse{
			// Tier 1: Essential
			{VerseID: "Matt.25.46", Importance: 1},  // Eternal punishment
			{VerseID: "Matt.25.41", Importance: 1},  // Depart, eternal fire
			{VerseID: "Rev.20.15", Importance: 1},   // Lake of fire
			{VerseID: "Rev.20.14", Importance: 1},   // Death and Hades thrown into lake
			{VerseID: "Mark.9.43", Importance: 1},   // Unquenchable fire
			{VerseID: "Mark.9.48", Importance: 1},   // Worm does not die, fire not quenched
			{VerseID: "2Thess.1.9", Importance: 1},  // Eternal destruction
			{VerseID: "Matt.10.28", Importance: 1},  // Fear him who can destroy soul and body
			{VerseID: "Luke.16.23", Importance: 1},  // In Hades, being in torment
			{VerseID: "Luke.16.24", Importance: 1},  // In anguish in this flame
			// Tier 2: Important
			{VerseID: "Rev.14.11", Importance: 2},   // Smoke of torment forever
			{VerseID: "Rev.21.8", Importance: 2},    // Lake that burns with fire
			{VerseID: "Matt.13.42", Importance: 2},  // Furnace of fire
			{VerseID: "Matt.13.50", Importance: 2},  // Furnace of fire
			{VerseID: "Matt.8.12", Importance: 2},   // Outer darkness
			{VerseID: "Matt.22.13", Importance: 2},  // Outer darkness
			{VerseID: "Matt.25.30", Importance: 2},  // Outer darkness
			{VerseID: "Jude.1.7", Importance: 2},    // Punishment of eternal fire
			{VerseID: "2Pet.2.4", Importance: 2},    // God did not spare angels
			{VerseID: "Heb.10.27", Importance: 2},   // Fearful expectation of judgment
			// Tier 3: Supporting
			{VerseID: "Luke.16.26", Importance: 3},  // Great chasm fixed
			{VerseID: "Matt.5.22", Importance: 3},   // Liable to hell of fire
			{VerseID: "Matt.5.29", Importance: 3},   // Whole body thrown into hell
			{VerseID: "Matt.23.33", Importance: 3},  // How will you escape hell
			{VerseID: "Isa.66.24", Importance: 3},   // Worm shall not die
			{VerseID: "Dan.12.2", Importance: 3},    // Everlasting contempt
			{VerseID: "John.3.36", Importance: 3},   // Wrath of God remains
			{VerseID: "Rev.19.20", Importance: 3},   // Thrown alive into lake of fire
		},
	},
	{
		Name:     "Sin",
		Slug:     "sin",
		Category: "concept",
		Description: "Transgression of God's law and falling short of His glory. " +
			"The universal human condition that separates us from God.",
		Verses: []CanonicalVerse{
			// Tier 1: Essential
			{VerseID: "Rom.3.23", Importance: 1},   // All have sinned
			{VerseID: "Rom.6.23", Importance: 1},   // Wages of sin is death
			{VerseID: "1John.3.4", Importance: 1},  // Sin is lawlessness
			{VerseID: "1John.1.8", Importance: 1},  // If we say we have no sin
			{VerseID: "1John.1.9", Importance: 1},  // If we confess our sins
			{VerseID: "Rom.5.12", Importance: 1},   // Sin entered through one man
			{VerseID: "Isa.59.2", Importance: 1},   // Sins have hidden his face
			{VerseID: "Isa.53.6", Importance: 1},   // All we like sheep have gone astray
			{VerseID: "James.4.17", Importance: 1}, // Knows good and does not do it
			{VerseID: "Rom.3.10", Importance: 1},   // None is righteous
			// Tier 2: Important
			{VerseID: "Rom.5.19", Importance: 2},   // By one man's disobedience
			{VerseID: "Rom.7.14", Importance: 2},   // Sold under sin
			{VerseID: "Rom.7.18", Importance: 2},   // Nothing good dwells in me
			{VerseID: "Rom.7.23", Importance: 2},   // Law of sin in my members
			{VerseID: "Gal.3.22", Importance: 2},   // Scripture imprisoned all under sin
			{VerseID: "Eccl.7.20", Importance: 2},  // No one who does good and never sins
			{VerseID: "Ps.51.5", Importance: 2},    // In sin did my mother conceive me
			{VerseID: "Gen.6.5", Importance: 2},    // Every intention of heart evil
			{VerseID: "Jer.17.9", Importance: 2},   // Heart is deceitful above all
			{VerseID: "Mark.7.21", Importance: 2},  // Out of the heart come evil thoughts
			// Tier 3: Supporting
			{VerseID: "Gen.3.6", Importance: 3},    // She took and ate
			{VerseID: "1John.2.16", Importance: 3}, // Lust of flesh, eyes, pride
			{VerseID: "James.1.14", Importance: 3}, // Tempted by own desires
			{VerseID: "James.1.15", Importance: 3}, // Desire gives birth to sin
			{VerseID: "Heb.3.13", Importance: 3},   // Hardened by deceitfulness of sin
			{VerseID: "Gal.5.19", Importance: 3},   // Works of the flesh
			{VerseID: "Eph.2.1", Importance: 3},    // Dead in trespasses and sins
			{VerseID: "Col.3.5", Importance: 3},    // Put to death what is earthly
		},
	},
	{
		Name:     "Forgiveness",
		Slug:     "forgiveness",
		Category: "concept",
		Description: "The act of pardoning offenses. God's forgiveness of our sins through Christ, " +
			"and our call to forgive others as we have been forgiven.",
		Verses: []CanonicalVerse{
			// Tier 1: Essential
			{VerseID: "Eph.1.7", Importance: 1},     // Forgiveness according to riches
			{VerseID: "Col.1.14", Importance: 1},    // Forgiveness of sins
			{VerseID: "1John.1.9", Importance: 1},   // Faithful to forgive
			{VerseID: "Ps.103.12", Importance: 1},   // As far as east from west
			{VerseID: "Matt.6.14", Importance: 1},   // If you forgive others
			{VerseID: "Matt.6.15", Importance: 1},   // If you do not forgive
			{VerseID: "Matt.18.21", Importance: 1},  // How many times forgive
			{VerseID: "Matt.18.22", Importance: 1},  // Seventy times seven
			{VerseID: "Eph.4.32", Importance: 1},    // Forgiving as God forgave you
			{VerseID: "Col.3.13", Importance: 1},    // Forgiving as Lord forgave you
			// Tier 2: Important
			{VerseID: "Acts.10.43", Importance: 2},  // Forgiveness through his name
			{VerseID: "Acts.13.38", Importance: 2},  // Forgiveness proclaimed
			{VerseID: "Isa.43.25", Importance: 2},   // I blot out your transgressions
			{VerseID: "Jer.31.34", Importance: 2},   // I will remember their sin no more
			{VerseID: "Mic.7.18", Importance: 2},    // Pardons iniquity
			{VerseID: "Heb.8.12", Importance: 2},    // Remember their sins no more
			{VerseID: "Heb.10.17", Importance: 2},   // Remember their sins no more
			{VerseID: "Mark.11.25", Importance: 2},  // When you stand praying, forgive
			{VerseID: "Luke.17.3", Importance: 2},   // If he repents, forgive
			{VerseID: "Luke.17.4", Importance: 2},   // Forgive him
			// Tier 3: Supporting
			{VerseID: "Matt.18.35", Importance: 3},  // Forgive brother from heart
			{VerseID: "Luke.6.37", Importance: 3},   // Forgive and you will be forgiven
			{VerseID: "Luke.23.34", Importance: 3},  // Father forgive them
			{VerseID: "Acts.7.60", Importance: 3},   // Lord do not hold this sin
			{VerseID: "2Cor.2.7", Importance: 3},    // Forgive and comfort
			{VerseID: "Ps.32.1", Importance: 3},     // Blessed whose transgression forgiven
			{VerseID: "Ps.32.5", Importance: 3},     // You forgave the iniquity
			{VerseID: "Ps.86.5", Importance: 3},     // Good and forgiving
		},
	},
	{
		Name:     "Love",
		Slug:     "love",
		Category: "concept",
		Description: "God's essential nature and the greatest commandment. Divine love expressed " +
			"in Christ's sacrifice and to be reflected in how believers love God and others.",
		Verses: []CanonicalVerse{
			// Tier 1: Essential
			{VerseID: "1John.4.8", Importance: 1},   // God is love
			{VerseID: "1John.4.16", Importance: 1},  // God is love
			{VerseID: "John.3.16", Importance: 1},   // God so loved the world
			{VerseID: "Rom.5.8", Importance: 1},     // God shows his love
			{VerseID: "1Cor.13.4", Importance: 1},   // Love is patient
			{VerseID: "1Cor.13.13", Importance: 1},  // Greatest of these is love
			{VerseID: "Matt.22.37", Importance: 1},  // Love the Lord your God
			{VerseID: "Matt.22.39", Importance: 1},  // Love your neighbor as yourself
			{VerseID: "John.13.34", Importance: 1},  // New commandment: love one another
			{VerseID: "John.13.35", Importance: 1},  // By this all will know
			// Tier 2: Important
			{VerseID: "1Cor.13.1", Importance: 2},   // If I speak but have not love
			{VerseID: "1Cor.13.2", Importance: 2},   // If I have faith but not love
			{VerseID: "1Cor.13.3", Importance: 2},   // If I give away all but not love
			{VerseID: "1Cor.13.5", Importance: 2},   // Not rude, not self-seeking
			{VerseID: "1Cor.13.6", Importance: 2},   // Rejoices with truth
			{VerseID: "1Cor.13.7", Importance: 2},   // Bears, believes, hopes, endures
			{VerseID: "1Cor.13.8", Importance: 2},   // Love never fails
			{VerseID: "1John.4.10", Importance: 2},  // Not that we loved God
			{VerseID: "1John.4.19", Importance: 2},  // We love because he first loved
			{VerseID: "John.15.13", Importance: 2},  // Greater love has no one
			// Tier 3: Supporting
			{VerseID: "1John.3.16", Importance: 3},  // He laid down his life
			{VerseID: "1John.3.18", Importance: 3},  // Not in word but in deed
			{VerseID: "1John.4.11", Importance: 3},  // We ought to love one another
			{VerseID: "1John.4.20", Importance: 3},  // Cannot love God, hate brother
			{VerseID: "1John.4.21", Importance: 3},  // Love brother also
			{VerseID: "Rom.13.10", Importance: 3},   // Love does no wrong
			{VerseID: "Gal.5.14", Importance: 3},    // Whole law fulfilled in love
			{VerseID: "Eph.5.25", Importance: 3},    // Husbands love your wives
			{VerseID: "Col.3.14", Importance: 3},    // Love binds together
			{VerseID: "1Pet.4.8", Importance: 3},    // Love covers multitude of sins
		},
	},
	{
		Name:     "Prayer",
		Slug:     "prayer",
		Category: "concept",
		Description: "Communication with God. Includes adoration, confession, thanksgiving, and " +
			"supplication. Jesus taught us how to pray and promised to answer.",
		Verses: []CanonicalVerse{
			// Tier 1: Essential
			{VerseID: "Matt.6.9", Importance: 1},    // Our Father in heaven
			{VerseID: "Matt.6.10", Importance: 1},   // Your kingdom come
			{VerseID: "Matt.6.11", Importance: 1},   // Give us this day
			{VerseID: "Matt.6.12", Importance: 1},   // Forgive us our debts
			{VerseID: "Matt.6.13", Importance: 1},   // Lead us not into temptation
			{VerseID: "Phil.4.6", Importance: 1},    // Do not be anxious, pray
			{VerseID: "Phil.4.7", Importance: 1},    // Peace of God will guard
			{VerseID: "1John.5.14", Importance: 1},  // If we ask according to will
			{VerseID: "John.14.13", Importance: 1},  // Whatever you ask in my name
			{VerseID: "John.14.14", Importance: 1},  // Ask anything in my name
			// Tier 2: Important
			{VerseID: "Matt.7.7", Importance: 2},    // Ask and it will be given
			{VerseID: "Matt.7.8", Importance: 2},    // Everyone who asks receives
			{VerseID: "James.5.16", Importance: 2},  // Prayer of righteous is powerful
			{VerseID: "1Thess.5.17", Importance: 2}, // Pray without ceasing
			{VerseID: "Col.4.2", Importance: 2},     // Continue steadfastly in prayer
			{VerseID: "Luke.18.1", Importance: 2},   // Always pray, not lose heart
			{VerseID: "Jer.33.3", Importance: 2},    // Call to me and I will answer
			{VerseID: "Matt.18.19", Importance: 2},  // If two agree
			{VerseID: "Matt.18.20", Importance: 2},  // Where two or three gathered
			{VerseID: "Rom.8.26", Importance: 2},    // Spirit helps us in prayer
			// Tier 3: Supporting
			{VerseID: "Matt.6.6", Importance: 3},    // Go into your room
			{VerseID: "Matt.21.22", Importance: 3},  // Whatever you ask in prayer
			{VerseID: "Mark.11.24", Importance: 3},  // Believe you have received
			{VerseID: "John.15.7", Importance: 3},   // Ask whatever you wish
			{VerseID: "John.16.24", Importance: 3},  // Ask and you will receive
			{VerseID: "James.1.5", Importance: 3},   // If any lacks wisdom, ask
			{VerseID: "James.4.2", Importance: 3},   // You do not have because
			{VerseID: "James.4.3", Importance: 3},   // You ask wrongly
			{VerseID: "1Pet.3.12", Importance: 3},   // Eyes of Lord on righteous
			{VerseID: "Ps.145.18", Importance: 3},   // Lord is near to all who call
		},
	},
	{
		Name:     "Worship",
		Slug:     "worship",
		Category: "concept",
		Description: "Reverence and adoration given to God alone. True worship is in spirit " +
			"and truth, involving the whole person in response to God's worth.",
		Verses: []CanonicalVerse{
			// Tier 1: Essential
			{VerseID: "John.4.23", Importance: 1},   // Worship in spirit and truth
			{VerseID: "John.4.24", Importance: 1},   // God is spirit
			{VerseID: "Ps.95.6", Importance: 1},     // Come let us worship and bow down
			{VerseID: "Rom.12.1", Importance: 1},    // Present bodies as living sacrifice
			{VerseID: "Exod.20.3", Importance: 1},   // No other gods before me
			{VerseID: "Exod.20.4", Importance: 1},   // No graven image
			{VerseID: "Matt.4.10", Importance: 1},   // Worship the Lord your God only
			{VerseID: "Rev.4.11", Importance: 1},    // Worthy are you our Lord
			{VerseID: "Ps.100.2", Importance: 1},    // Serve the Lord with gladness
			{VerseID: "Heb.12.28", Importance: 1},   // Offer acceptable worship
			// Tier 2: Important
			{VerseID: "Ps.29.2", Importance: 2},     // Worship in the splendor of holiness
			{VerseID: "Ps.96.9", Importance: 2},     // Worship in the splendor of holiness
			{VerseID: "Ps.99.5", Importance: 2},     // Worship at his footstool
			{VerseID: "Rev.5.12", Importance: 2},    // Worthy is the Lamb
			{VerseID: "Rev.5.13", Importance: 2},    // To him be blessing and honor
			{VerseID: "Ps.95.1", Importance: 2},     // Come let us sing to the Lord
			{VerseID: "Ps.100.4", Importance: 2},    // Enter his gates with thanksgiving
			{VerseID: "Col.3.16", Importance: 2},    // Singing psalms hymns spiritual songs
			{VerseID: "Eph.5.19", Importance: 2},    // Speaking to one another in psalms
			{VerseID: "Ps.150.6", Importance: 2},    // Let everything that breathes praise
			// Tier 3: Supporting
			{VerseID: "Ps.34.3", Importance: 3},     // Magnify the Lord with me
			{VerseID: "Ps.63.4", Importance: 3},     // I will bless you as long as I live
			{VerseID: "Ps.103.1", Importance: 3},    // Bless the Lord O my soul
			{VerseID: "Ps.145.3", Importance: 3},    // Great is the Lord
			{VerseID: "Isa.6.3", Importance: 3},     // Holy holy holy
			{VerseID: "Rev.4.8", Importance: 3},     // Holy holy holy
			{VerseID: "Ps.122.1", Importance: 3},    // I was glad when they said
			{VerseID: "Heb.10.25", Importance: 3},   // Not neglecting to meet together
		},
	},
	{
		Name:     "Baptism",
		Slug:     "baptism",
		Category: "concept",
		Description: "The ordinance of water baptism symbolizing identification with Christ's death, " +
			"burial, and resurrection. An outward sign of inward faith.",
		Verses: []CanonicalVerse{
			// Tier 1: Essential
			{VerseID: "Matt.28.19", Importance: 1},  // Baptizing in the name
			{VerseID: "Matt.3.16", Importance: 1},   // Jesus baptized
			{VerseID: "Mark.16.16", Importance: 1},  // Whoever believes and is baptized
			{VerseID: "Acts.2.38", Importance: 1},   // Repent and be baptized
			{VerseID: "Acts.2.41", Importance: 1},   // Those who received word were baptized
			{VerseID: "Rom.6.3", Importance: 1},     // Baptized into his death
			{VerseID: "Rom.6.4", Importance: 1},     // Buried with him through baptism
			{VerseID: "Gal.3.27", Importance: 1},    // Baptized into Christ
			{VerseID: "Col.2.12", Importance: 1},    // Buried with him in baptism
			{VerseID: "1Pet.3.21", Importance: 1},   // Baptism now saves you
			// Tier 2: Important
			{VerseID: "Acts.8.36", Importance: 2},   // What prevents me from being baptized
			{VerseID: "Acts.8.38", Importance: 2},   // They went down into water
			{VerseID: "Acts.10.47", Importance: 2},  // Can anyone withhold water
			{VerseID: "Acts.10.48", Importance: 2},  // Commanded them to be baptized
			{VerseID: "Acts.16.33", Importance: 2},  // Baptized immediately
			{VerseID: "Acts.22.16", Importance: 2},  // Arise and be baptized
			{VerseID: "1Cor.12.13", Importance: 2},  // By one Spirit baptized into one body
			{VerseID: "Eph.4.5", Importance: 2},     // One Lord, one faith, one baptism
			{VerseID: "Matt.3.11", Importance: 2},   // I baptize with water
			{VerseID: "John.3.5", Importance: 2},    // Born of water and Spirit
			// Tier 3: Supporting
			{VerseID: "Acts.8.12", Importance: 3},   // Believed and were baptized
			{VerseID: "Acts.9.18", Importance: 3},   // Paul baptized
			{VerseID: "Acts.16.15", Importance: 3},  // Lydia baptized
			{VerseID: "Acts.18.8", Importance: 3},   // Corinthians believed and baptized
			{VerseID: "Acts.19.5", Importance: 3},   // Baptized in name of Lord Jesus
			{VerseID: "1Cor.1.13", Importance: 3},   // Were you baptized in Paul's name
			{VerseID: "1Cor.1.14", Importance: 3},   // I baptized none of you
			{VerseID: "Titus.3.5", Importance: 3},   // Washing of regeneration
		},
	},
	{
		Name:     "Communion",
		Slug:     "communion",
		Category: "concept",
		Description: "The Lord's Supper, instituted by Christ. A memorial of His death, " +
			"a proclamation of the gospel, and a fellowship of believers until He returns.",
		Verses: []CanonicalVerse{
			// Tier 1: Essential
			{VerseID: "Matt.26.26", Importance: 1},   // This is my body
			{VerseID: "Matt.26.27", Importance: 1},   // Drink of it, all of you
			{VerseID: "Matt.26.28", Importance: 1},   // This is my blood of the covenant
			{VerseID: "1Cor.11.23", Importance: 1},   // I received from the Lord
			{VerseID: "1Cor.11.24", Importance: 1},   // Do this in remembrance
			{VerseID: "1Cor.11.25", Importance: 1},   // This cup is the new covenant
			{VerseID: "1Cor.11.26", Importance: 1},   // Proclaim the Lord's death
			{VerseID: "Luke.22.19", Importance: 1},   // Do this in remembrance of me
			{VerseID: "Luke.22.20", Importance: 1},   // New covenant in my blood
			{VerseID: "1Cor.10.16", Importance: 1},   // Cup of blessing, participation
			// Tier 2: Important
			{VerseID: "1Cor.11.27", Importance: 2},   // Unworthy manner
			{VerseID: "1Cor.11.28", Importance: 2},   // Examine himself
			{VerseID: "1Cor.11.29", Importance: 2},   // Discerning the body
			{VerseID: "1Cor.10.17", Importance: 2},   // One bread, one body
			{VerseID: "1Cor.10.21", Importance: 2},   // Cannot drink cup of Lord and demons
			{VerseID: "Mark.14.22", Importance: 2},   // Take, this is my body
			{VerseID: "Mark.14.23", Importance: 2},   // Took a cup
			{VerseID: "Mark.14.24", Importance: 2},   // Blood of the covenant
			{VerseID: "John.6.53", Importance: 2},    // Eat flesh, drink blood
			{VerseID: "John.6.54", Importance: 2},    // Eternal life
			// Tier 3: Supporting
			{VerseID: "John.6.55", Importance: 3},    // True food, true drink
			{VerseID: "John.6.56", Importance: 3},    // Abides in me and I in him
			{VerseID: "Acts.2.42", Importance: 3},    // Breaking of bread
			{VerseID: "Acts.2.46", Importance: 3},    // Breaking bread from house to house
			{VerseID: "Acts.20.7", Importance: 3},    // First day, to break bread
			{VerseID: "1Cor.5.7", Importance: 3},     // Christ our Passover
			{VerseID: "1Cor.5.8", Importance: 3},     // Let us celebrate the festival
			{VerseID: "Heb.9.20", Importance: 3},     // Blood of the covenant
		},
	},
	{
		Name:     "Marriage",
		Slug:     "marriage",
		Category: "concept",
		Description: "The covenant union between one man and one woman, instituted by God at creation. " +
			"A picture of Christ and the church.",
		Verses: []CanonicalVerse{
			// Tier 1: Essential
			{VerseID: "Gen.2.24", Importance: 1},    // Man shall leave and cleave
			{VerseID: "Matt.19.4", Importance: 1},   // Made them male and female
			{VerseID: "Matt.19.5", Importance: 1},   // Two shall become one flesh
			{VerseID: "Matt.19.6", Importance: 1},   // What God has joined together
			{VerseID: "Eph.5.25", Importance: 1},    // Husbands love your wives
			{VerseID: "Eph.5.31", Importance: 1},    // Two shall become one flesh
			{VerseID: "Eph.5.32", Importance: 1},    // Mystery: Christ and church
			{VerseID: "Eph.5.22", Importance: 1},    // Wives submit to husbands
			{VerseID: "Eph.5.33", Importance: 1},    // Love wife, respect husband
			{VerseID: "Heb.13.4", Importance: 1},    // Marriage bed undefiled
			// Tier 2: Important
			{VerseID: "Gen.2.18", Importance: 2},    // Not good to be alone
			{VerseID: "Gen.2.22", Importance: 2},    // Made a woman
			{VerseID: "Gen.2.23", Importance: 2},    // Bone of my bones
			{VerseID: "Prov.18.22", Importance: 2},  // Finds a wife finds good thing
			{VerseID: "Prov.31.10", Importance: 2},  // Excellent wife
			{VerseID: "1Cor.7.2", Importance: 2},    // Each man have own wife
			{VerseID: "1Cor.7.3", Importance: 2},    // Husband give conjugal rights
			{VerseID: "1Cor.7.4", Importance: 2},    // Authority over each other's body
			{VerseID: "Col.3.18", Importance: 2},    // Wives submit
			{VerseID: "Col.3.19", Importance: 2},    // Husbands love, not harsh
			// Tier 3: Supporting
			{VerseID: "1Pet.3.1", Importance: 3},    // Wives submit
			{VerseID: "1Pet.3.7", Importance: 3},    // Husbands live with understanding
			{VerseID: "Mal.2.14", Importance: 3},    // Wife by covenant
			{VerseID: "Mal.2.15", Importance: 3},    // Guard yourself in your spirit
			{VerseID: "Mal.2.16", Importance: 3},    // God hates divorce
			{VerseID: "1Cor.7.10", Importance: 3},   // Wife should not separate
			{VerseID: "1Cor.7.11", Importance: 3},   // Husband not divorce
			{VerseID: "Mark.10.9", Importance: 3},   // What God has joined
		},
	},
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

	fmt.Printf("Inserting %d core topics...\n\n", len(CoreTopics))

	totalTopics := 0
	totalVerses := 0

	for _, topic := range CoreTopics {
		topicID, verseCount, err := insertTopic(ctx, db, topic)
		if err != nil {
			fmt.Printf("❌ Failed to insert %s: %v\n", topic.Name, err)
			continue
		}
		fmt.Printf("✅ %s (ID: %d) - %d verses\n", topic.Name, topicID, verseCount)
		totalTopics++
		totalVerses += verseCount
	}

	fmt.Printf("\n" + "=" + string(make([]byte, 50)) + "\n")
	fmt.Printf("SUMMARY\n")
	fmt.Printf("=" + string(make([]byte, 50)) + "\n")
	fmt.Printf("Topics created: %d/%d\n", totalTopics, len(CoreTopics))
	fmt.Printf("Total verses mapped: %d\n", totalVerses)
	fmt.Println("\nRefreshing materialized view...")

	_, err = db.ExecContext(ctx, "REFRESH MATERIALIZED VIEW api_views.mv_topics_summary")
	if err != nil {
		fmt.Printf("Warning: Failed to refresh materialized view: %v\n", err)
	} else {
		fmt.Println("Done!")
	}
}

func insertTopic(ctx context.Context, db *sqlx.DB, topic TopicDefinition) (int, int, error) {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert topic
	var topicID int
	insertTopicSQL := `
		INSERT INTO api.topics (name, slug, source, topic, sub_topic, category, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	err = tx.QueryRowContext(ctx, insertTopicSQL,
		topic.Name,
		topic.Slug,
		"claude_4.5_opus",
		topic.Name,
		"",
		topic.Category,
		topic.Description,
	).Scan(&topicID)
	if err != nil {
		return 0, 0, fmt.Errorf("insert topic: %w", err)
	}

	// Get verse IDs
	verseOSISIDs := make([]string, len(topic.Verses))
	for i, v := range topic.Verses {
		verseOSISIDs[i] = v.VerseID
	}

	query := `SELECT id, osis_verse_id FROM api.verses WHERE osis_verse_id = ANY($1)`
	rows, err := tx.QueryContext(ctx, query, pq.Array(verseOSISIDs))
	if err != nil {
		return 0, 0, fmt.Errorf("query verses: %w", err)
	}

	verseIDMap := make(map[string]int)
	for rows.Next() {
		var id int
		var osisID string
		if err := rows.Scan(&id, &osisID); err != nil {
			rows.Close()
			return 0, 0, fmt.Errorf("scan verse: %w", err)
		}
		verseIDMap[osisID] = id
	}
	rows.Close()

	// Insert topic_verses mappings
	insertMappingSQL := `INSERT INTO api.topic_verses (topic_id, verse_id) VALUES ($1, $2)`
	insertedCount := 0

	for _, cv := range topic.Verses {
		verseID, ok := verseIDMap[cv.VerseID]
		if !ok {
			continue
		}

		_, err := tx.ExecContext(ctx, insertMappingSQL, topicID, verseID)
		if err != nil {
			continue
		}
		insertedCount++
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("commit: %w", err)
	}

	return topicID, insertedCount, nil
}
