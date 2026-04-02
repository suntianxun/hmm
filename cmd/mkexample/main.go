package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/parquet-go/parquet-go"
)

type Record struct {
	ID            int64   `parquet:"id"`
	FirstName     string  `parquet:"first_name"`
	LastName      string  `parquet:"last_name"`
	Email         string  `parquet:"email"`
	Age           int64   `parquet:"age"`
	City          string  `parquet:"city"`
	State         string  `parquet:"state"`
	Country       string  `parquet:"country"`
	PostalCode    string  `parquet:"postal_code"`
	Phone         string  `parquet:"phone"`
	Department    string  `parquet:"department"`
	JobTitle      string  `parquet:"job_title"`
	Company       string  `parquet:"company"`
	Salary        float64 `parquet:"salary"`
	YearsExp      int64   `parquet:"years_experience"`
	Rating        float64 `parquet:"rating"`
	IsActive      bool    `parquet:"is_active"`
	SignupDate    string  `parquet:"signup_date"`
	LastLogin     string  `parquet:"last_login"`
	LoginCount    int64   `parquet:"login_count"`
	Plan          string  `parquet:"plan"`
	StorageUsedMB float64 `parquet:"storage_used_mb"`
	ProjectCount  int64   `parquet:"project_count"`
	TeamSize      int64   `parquet:"team_size"`
	Region        string  `parquet:"region"`
	Language      string  `parquet:"language"`
	OS            string  `parquet:"os"`
	Browser       string  `parquet:"browser"`
	Referral      string  `parquet:"referral_source"`
	NPS           int64   `parquet:"nps_score"`
}

var (
	firstNames  = []string{"Alice", "Bob", "Charlie", "Diana", "Eve", "Frank", "Grace", "Hank", "Iris", "Jack", "Karen", "Leo", "Mona", "Nate", "Olivia", "Pete", "Quinn", "Rosa", "Sam", "Tina", "Uma", "Vic", "Wendy", "Xander", "Yara", "Zane"}
	lastNames   = []string{"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis", "Rodriguez", "Martinez", "Hernandez", "Lopez", "Gonzalez", "Wilson", "Anderson", "Thomas", "Taylor", "Moore", "Jackson", "Martin"}
	cities      = []string{"New York", "Los Angeles", "Chicago", "Houston", "Phoenix", "Philadelphia", "San Antonio", "San Diego", "Dallas", "Austin", "Seattle", "Denver", "Boston", "Nashville", "Portland", "Miami", "Atlanta", "Minneapolis", "Detroit", "San Francisco"}
	states      = []string{"NY", "CA", "IL", "TX", "AZ", "PA", "TX", "CA", "TX", "TX", "WA", "CO", "MA", "TN", "OR", "FL", "GA", "MN", "MI", "CA"}
	countries   = []string{"US", "US", "US", "US", "US", "CA", "CA", "UK", "UK", "DE"}
	departments = []string{"Engineering", "Marketing", "Sales", "Design", "Product", "Finance", "HR", "Legal", "Operations", "Support"}
	jobTitles   = []string{"Engineer", "Senior Engineer", "Staff Engineer", "Manager", "Director", "VP", "Analyst", "Coordinator", "Specialist", "Lead", "Architect", "Consultant"}
	companies   = []string{"Acme Corp", "Globex", "Initech", "Umbrella", "Stark Industries", "Wayne Enterprises", "Cyberdyne", "Soylent Corp", "Tyrell Corp", "Weyland-Yutani", "Aperture Science", "Black Mesa", "Oscorp", "LexCorp", "Massive Dynamic"}
	plans       = []string{"Free", "Starter", "Pro", "Business", "Enterprise"}
	regions     = []string{"North America", "Europe", "Asia Pacific", "Latin America", "Middle East"}
	languages   = []string{"English", "Spanish", "French", "German", "Portuguese", "Japanese", "Korean", "Chinese", "Hindi", "Arabic"}
	oses        = []string{"macOS", "Windows", "Linux", "ChromeOS", "iOS", "Android"}
	browsers    = []string{"Chrome", "Firefox", "Safari", "Edge", "Arc", "Brave"}
	referrals   = []string{"Google", "Twitter", "LinkedIn", "Friend", "Blog", "Conference", "YouTube", "Reddit", "HackerNews", "Direct"}
)

func main() {
	r := rand.New(rand.NewSource(42))

	// Small example
	writeSmall()

	// Large example
	path := "/tmp/example_large.parquet"
	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}

	w := parquet.NewGenericWriter[Record](f)

	const totalRows = 500_000
	const batchSize = 10_000
	batch := make([]Record, batchSize)

	baseDate := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < totalRows; i++ {
		first := firstNames[r.Intn(len(firstNames))]
		last := lastNames[r.Intn(len(lastNames))]
		cityIdx := r.Intn(len(cities))

		signup := baseDate.Add(time.Duration(r.Intn(1500)) * 24 * time.Hour)
		lastLogin := signup.Add(time.Duration(r.Intn(500)+1) * 24 * time.Hour)

		batch[i%batchSize] = Record{
			ID:            int64(i + 1),
			FirstName:     first,
			LastName:      last,
			Email:         fmt.Sprintf("%s.%s%d@example.com", first, last, i),
			Age:           int64(22 + r.Intn(43)),
			City:          cities[cityIdx],
			State:         states[cityIdx%len(states)],
			Country:       countries[r.Intn(len(countries))],
			PostalCode:    fmt.Sprintf("%05d", 10000+r.Intn(90000)),
			Phone:         fmt.Sprintf("+1-%03d-%03d-%04d", r.Intn(900)+100, r.Intn(900)+100, r.Intn(9000)+1000),
			Department:    departments[r.Intn(len(departments))],
			JobTitle:      jobTitles[r.Intn(len(jobTitles))],
			Company:       companies[r.Intn(len(companies))],
			Salary:        float64(40000+r.Intn(160000)) + float64(r.Intn(100))/100,
			YearsExp:      int64(r.Intn(30)),
			Rating:        float64(r.Intn(40)+10) / 10.0,
			IsActive:      r.Float64() > 0.15,
			SignupDate:    signup.Format("2006-01-02"),
			LastLogin:     lastLogin.Format("2006-01-02"),
			LoginCount:    int64(r.Intn(5000)),
			Plan:          plans[r.Intn(len(plans))],
			StorageUsedMB: float64(r.Intn(100000)) / 10.0,
			ProjectCount:  int64(r.Intn(50)),
			TeamSize:      int64(1 + r.Intn(50)),
			Region:        regions[r.Intn(len(regions))],
			Language:      languages[r.Intn(len(languages))],
			OS:            oses[r.Intn(len(oses))],
			Browser:       browsers[r.Intn(len(browsers))],
			Referral:      referrals[r.Intn(len(referrals))],
			NPS:           int64(r.Intn(11)),
		}

		if (i+1)%batchSize == 0 {
			if _, err := w.Write(batch); err != nil {
				log.Fatal(err)
			}
		}
	}

	if err := w.Close(); err != nil {
		log.Fatal(err)
	}

	log.Printf("Wrote %d records (30 columns) to %s", totalRows, path)
}

func writeSmall() {
	type SmallRecord struct {
		Name       string  `parquet:"name"`
		Age        int64   `parquet:"age"`
		City       string  `parquet:"city"`
		Score      float64 `parquet:"score"`
		Department string  `parquet:"department"`
	}

	path := "/tmp/example.parquet"
	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}

	w := parquet.NewGenericWriter[SmallRecord](f)
	records := []SmallRecord{
		{"Alice", 30, "New York", 95.5, "Engineering"},
		{"Bob", 25, "San Francisco", 88.2, "Marketing"},
		{"Charlie", 35, "Chicago", 92.1, "Engineering"},
		{"Diana", 28, "Austin", 97.8, "Design"},
		{"Eve", 32, "Seattle", 85.0, "Marketing"},
		{"Frank", 41, "Denver", 91.3, "Engineering"},
		{"Grace", 27, "Boston", 89.7, "Design"},
		{"Hank", 38, "Portland", 78.4, "Sales"},
		{"Iris", 29, "Miami", 94.6, "Engineering"},
		{"Jack", 33, "Nashville", 82.9, "Sales"},
		{"Karen", 26, "Atlanta", 96.1, "Design"},
		{"Leo", 45, "Detroit", 87.5, "Marketing"},
		{"Mona", 31, "Phoenix", 90.0, "Engineering"},
		{"Nate", 37, "Dallas", 83.7, "Sales"},
		{"Olivia", 24, "San Diego", 98.2, "Design"},
	}

	if _, err := w.Write(records); err != nil {
		log.Fatal(err)
	}
	if err := w.Close(); err != nil {
		log.Fatal(err)
	}
	log.Printf("Wrote %d records to %s", len(records), path)
}
