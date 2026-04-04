package main

import (
	"fmt"
	"log"
	"math/rand"

	"github.com/xuri/excelize/v2"
)

func main() {
	f := excelize.NewFile()
	r := rand.New(rand.NewSource(42))

	// Sheet 1: Employees
	f.SetSheetName("Sheet1", "Employees")
	empHeaders := []string{"ID", "Name", "Age", "Department", "Salary", "City"}
	names := []string{"Alice", "Bob", "Charlie", "Diana", "Eve", "Frank", "Grace", "Hank", "Iris", "Jack", "Karen", "Leo", "Mona", "Nate", "Olivia"}
	depts := []string{"Engineering", "Marketing", "Sales", "Design", "Product"}
	cities := []string{"New York", "San Francisco", "Chicago", "Austin", "Seattle", "Denver", "Boston", "Portland"}
	for i, h := range empHeaders {
		f.SetCellValue("Employees", cell(1, i+1), h)
	}
	for i := 0; i < 50; i++ {
		row := i + 2
		f.SetCellValue("Employees", cell(row, 1), i+1)
		f.SetCellValue("Employees", cell(row, 2), names[r.Intn(len(names))])
		f.SetCellValue("Employees", cell(row, 3), 22+r.Intn(40))
		f.SetCellValue("Employees", cell(row, 4), depts[r.Intn(len(depts))])
		f.SetCellValue("Employees", cell(row, 5), 40000+r.Intn(120000))
		f.SetCellValue("Employees", cell(row, 6), cities[r.Intn(len(cities))])
	}

	// Sheet 2: Products
	f.NewSheet("Products")
	prodHeaders := []string{"SKU", "Product", "Category", "Price", "Stock", "Rating"}
	products := []string{"Widget A", "Widget B", "Gadget X", "Gadget Y", "Gizmo Pro", "Gizmo Lite", "Thingamajig", "Doohickey", "Whatchamacallit", "Contraption"}
	categories := []string{"Electronics", "Home", "Office", "Outdoor", "Kitchen"}
	for i, h := range prodHeaders {
		f.SetCellValue("Products", cell(1, i+1), h)
	}
	for i := 0; i < 30; i++ {
		row := i + 2
		f.SetCellValue("Products", cell(row, 1), fmt.Sprintf("SKU-%04d", i+1))
		f.SetCellValue("Products", cell(row, 2), products[r.Intn(len(products))])
		f.SetCellValue("Products", cell(row, 3), categories[r.Intn(len(categories))])
		f.SetCellValue("Products", cell(row, 4), fmt.Sprintf("%.2f", 9.99+float64(r.Intn(49000))/100))
		f.SetCellValue("Products", cell(row, 5), r.Intn(500))
		f.SetCellValue("Products", cell(row, 6), fmt.Sprintf("%.1f", 1.0+float64(r.Intn(40))/10))
	}

	// Sheet 3: Orders
	f.NewSheet("Orders")
	orderHeaders := []string{"OrderID", "Customer", "Product", "Quantity", "Total", "Date", "Status"}
	statuses := []string{"Pending", "Shipped", "Delivered", "Returned", "Cancelled"}
	for i, h := range orderHeaders {
		f.SetCellValue("Orders", cell(1, i+1), h)
	}
	for i := 0; i < 80; i++ {
		row := i + 2
		qty := 1 + r.Intn(10)
		price := 9.99 + float64(r.Intn(49000))/100
		f.SetCellValue("Orders", cell(row, 1), fmt.Sprintf("ORD-%06d", 1000+i))
		f.SetCellValue("Orders", cell(row, 2), names[r.Intn(len(names))])
		f.SetCellValue("Orders", cell(row, 3), products[r.Intn(len(products))])
		f.SetCellValue("Orders", cell(row, 4), qty)
		f.SetCellValue("Orders", cell(row, 5), fmt.Sprintf("%.2f", float64(qty)*price))
		f.SetCellValue("Orders", cell(row, 6), fmt.Sprintf("2025-%02d-%02d", 1+r.Intn(12), 1+r.Intn(28)))
		f.SetCellValue("Orders", cell(row, 7), statuses[r.Intn(len(statuses))])
	}

	// Sheet 4: Metrics
	f.NewSheet("Metrics")
	metricHeaders := []string{"Month", "Revenue", "Expenses", "Profit", "Customers", "Churn Rate"}
	months := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	for i, h := range metricHeaders {
		f.SetCellValue("Metrics", cell(1, i+1), h)
	}
	for i, month := range months {
		row := i + 2
		revenue := 100000 + r.Intn(200000)
		expenses := 60000 + r.Intn(80000)
		f.SetCellValue("Metrics", cell(row, 1), month)
		f.SetCellValue("Metrics", cell(row, 2), revenue)
		f.SetCellValue("Metrics", cell(row, 3), expenses)
		f.SetCellValue("Metrics", cell(row, 4), revenue-expenses)
		f.SetCellValue("Metrics", cell(row, 5), 500+r.Intn(1000))
		f.SetCellValue("Metrics", cell(row, 6), fmt.Sprintf("%.1f%%", 1.0+float64(r.Intn(80))/10))
	}

	// Sheet 5: Config
	f.NewSheet("Config")
	configHeaders := []string{"Key", "Value", "Description", "Updated"}
	keys := []string{"max_retries", "timeout_ms", "log_level", "feature_flag_v2", "cache_ttl", "batch_size", "rate_limit", "db_pool_size", "enable_metrics", "debug_mode"}
	values := []string{"3", "5000", "info", "true", "3600", "100", "1000", "20", "true", "false"}
	descs := []string{"Max retry attempts", "Request timeout in ms", "Logging level", "Enable v2 features", "Cache TTL in seconds", "Batch processing size", "Requests per minute", "DB connection pool", "Enable metrics collection", "Debug mode toggle"}
	for i, h := range configHeaders {
		f.SetCellValue("Config", cell(1, i+1), h)
	}
	for i := 0; i < len(keys); i++ {
		row := i + 2
		f.SetCellValue("Config", cell(row, 1), keys[i])
		f.SetCellValue("Config", cell(row, 2), values[i])
		f.SetCellValue("Config", cell(row, 3), descs[i])
		f.SetCellValue("Config", cell(row, 4), fmt.Sprintf("2025-%02d-%02d", 1+r.Intn(12), 1+r.Intn(28)))
	}

	path := "/Users/stephen/data/example.xlsx"
	if err := f.SaveAs(path); err != nil {
		log.Fatal(err)
	}
	log.Printf("Wrote 5-sheet Excel file to %s", path)
}

func cell(row, col int) string {
	colLetter := string(rune('A' + col - 1))
	return fmt.Sprintf("%s%d", colLetter, row)
}
