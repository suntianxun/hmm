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
	sw := make(map[string]*excelize.StreamWriter)

	names := []string{"Alice", "Bob", "Charlie", "Diana", "Eve", "Frank", "Grace", "Hank", "Iris", "Jack", "Karen", "Leo", "Mona", "Nate", "Olivia", "Pete", "Quinn", "Rosa", "Sam", "Tina"}
	depts := []string{"Engineering", "Marketing", "Sales", "Design", "Product", "Finance", "HR", "Legal", "Operations", "Support"}
	cities := []string{"New York", "San Francisco", "Chicago", "Austin", "Seattle", "Denver", "Boston", "Portland", "Miami", "Atlanta", "Dallas", "Phoenix", "Nashville", "Minneapolis", "Detroit"}
	statuses := []string{"Pending", "Shipped", "Delivered", "Returned", "Cancelled", "Processing", "On Hold"}
	products := []string{"Widget A", "Widget B", "Gadget X", "Gadget Y", "Gizmo Pro", "Gizmo Lite", "Thingamajig", "Doohickey", "Whatchamacallit", "Contraption", "Module Z", "Component K", "Device M", "Sensor N", "Adapter P"}
	categories := []string{"Electronics", "Home", "Office", "Outdoor", "Kitchen", "Automotive", "Industrial", "Medical"}
	regions := []string{"North America", "Europe", "Asia Pacific", "Latin America", "Middle East", "Africa"}
	plans := []string{"Free", "Starter", "Pro", "Business", "Enterprise"}
	actions := []string{"login", "logout", "page_view", "click", "purchase", "signup", "search", "download", "upload", "share", "comment", "like", "report", "settings_change", "api_call"}

	// Sheet 1: Employees — 100k rows
	sheetName := "Employees"
	f.SetSheetName("Sheet1", sheetName)
	w, _ := f.NewStreamWriter(sheetName)
	sw[sheetName] = w
	w.SetRow("A1", []interface{}{"ID", "FirstName", "LastName", "Age", "Department", "City", "Salary", "YearsExp", "Rating", "IsActive"})
	for i := 0; i < 100_000; i++ {
		cell, _ := excelize.CoordinatesToCellName(1, i+2)
		w.SetRow(cell, []interface{}{
			i + 1,
			names[r.Intn(len(names))],
			fmt.Sprintf("Last%d", r.Intn(500)),
			22 + r.Intn(43),
			depts[r.Intn(len(depts))],
			cities[r.Intn(len(cities))],
			40000 + r.Intn(160000),
			r.Intn(30),
			fmt.Sprintf("%.1f", 1.0+float64(r.Intn(40))/10),
			r.Float64() > 0.15,
		})
	}
	w.Flush()

	// Sheet 2: Orders — 200k rows
	sheetName = "Orders"
	f.NewSheet(sheetName)
	w, _ = f.NewStreamWriter(sheetName)
	sw[sheetName] = w
	w.SetRow("A1", []interface{}{"OrderID", "Customer", "Product", "Category", "Quantity", "UnitPrice", "Total", "Date", "Status", "Region"})
	for i := 0; i < 200_000; i++ {
		cell, _ := excelize.CoordinatesToCellName(1, i+2)
		qty := 1 + r.Intn(20)
		price := 5.0 + float64(r.Intn(50000))/100
		w.SetRow(cell, []interface{}{
			fmt.Sprintf("ORD-%07d", i+1),
			names[r.Intn(len(names))] + " " + fmt.Sprintf("Last%d", r.Intn(500)),
			products[r.Intn(len(products))],
			categories[r.Intn(len(categories))],
			qty,
			fmt.Sprintf("%.2f", price),
			fmt.Sprintf("%.2f", float64(qty)*price),
			fmt.Sprintf("2024-%02d-%02d", 1+r.Intn(12), 1+r.Intn(28)),
			statuses[r.Intn(len(statuses))],
			regions[r.Intn(len(regions))],
		})
	}
	w.Flush()

	// Sheet 3: EventLog — 300k rows
	sheetName = "EventLog"
	f.NewSheet(sheetName)
	w, _ = f.NewStreamWriter(sheetName)
	sw[sheetName] = w
	w.SetRow("A1", []interface{}{"EventID", "Timestamp", "UserID", "Action", "Page", "Duration_ms", "StatusCode", "IPAddress"})
	for i := 0; i < 300_000; i++ {
		cell, _ := excelize.CoordinatesToCellName(1, i+2)
		w.SetRow(cell, []interface{}{
			i + 1,
			fmt.Sprintf("2024-%02d-%02d %02d:%02d:%02d", 1+r.Intn(12), 1+r.Intn(28), r.Intn(24), r.Intn(60), r.Intn(60)),
			fmt.Sprintf("USR-%06d", r.Intn(50000)),
			actions[r.Intn(len(actions))],
			fmt.Sprintf("/page/%d", r.Intn(200)),
			r.Intn(30000),
			[]int{200, 200, 200, 200, 301, 302, 400, 401, 403, 404, 500}[r.Intn(11)],
			fmt.Sprintf("%d.%d.%d.%d", r.Intn(256), r.Intn(256), r.Intn(256), r.Intn(256)),
		})
	}
	w.Flush()

	// Sheet 4: Inventory — 50k rows
	sheetName = "Inventory"
	f.NewSheet(sheetName)
	w, _ = f.NewStreamWriter(sheetName)
	sw[sheetName] = w
	w.SetRow("A1", []interface{}{"SKU", "Product", "Category", "Warehouse", "Quantity", "MinStock", "UnitCost", "LastRestocked", "Supplier"})
	warehouses := []string{"WH-East", "WH-West", "WH-Central", "WH-South", "WH-North", "WH-EU", "WH-APAC"}
	suppliers := []string{"SupplyCo", "MegaParts", "GlobalSource", "DirectShip", "PrimeMfg", "ValueSupply", "QuickParts", "BulkDeal"}
	for i := 0; i < 50_000; i++ {
		cell, _ := excelize.CoordinatesToCellName(1, i+2)
		w.SetRow(cell, []interface{}{
			fmt.Sprintf("SKU-%06d", i+1),
			products[r.Intn(len(products))] + fmt.Sprintf(" v%d", r.Intn(10)),
			categories[r.Intn(len(categories))],
			warehouses[r.Intn(len(warehouses))],
			r.Intn(10000),
			10 + r.Intn(200),
			fmt.Sprintf("%.2f", 1.0+float64(r.Intn(50000))/100),
			fmt.Sprintf("2024-%02d-%02d", 1+r.Intn(12), 1+r.Intn(28)),
			suppliers[r.Intn(len(suppliers))],
		})
	}
	w.Flush()

	// Sheet 5: Metrics — 150k rows
	sheetName = "Metrics"
	f.NewSheet(sheetName)
	w, _ = f.NewStreamWriter(sheetName)
	sw[sheetName] = w
	w.SetRow("A1", []interface{}{"Timestamp", "Service", "Region", "CPU_pct", "Memory_MB", "Requests", "Errors", "Latency_p50", "Latency_p99", "Plan"})
	services := []string{"api-gateway", "auth-service", "user-service", "payment-service", "notification-service", "search-service", "analytics-engine", "data-pipeline"}
	for i := 0; i < 150_000; i++ {
		cell, _ := excelize.CoordinatesToCellName(1, i+2)
		w.SetRow(cell, []interface{}{
			fmt.Sprintf("2024-%02d-%02d %02d:%02d:00", 1+r.Intn(12), 1+r.Intn(28), r.Intn(24), r.Intn(60)),
			services[r.Intn(len(services))],
			regions[r.Intn(len(regions))],
			fmt.Sprintf("%.1f", float64(r.Intn(1000))/10),
			256 + r.Intn(7680),
			r.Intn(50000),
			r.Intn(500),
			fmt.Sprintf("%.1f", float64(r.Intn(2000))/10),
			fmt.Sprintf("%.1f", float64(50+r.Intn(9500))/10),
			plans[r.Intn(len(plans))],
		})
	}
	w.Flush()

	path := "/Users/stephen/data/large.xlsx"
	if err := f.SaveAs(path); err != nil {
		log.Fatal(err)
	}
	log.Printf("Wrote large Excel file to %s (100k+200k+300k+50k+150k = 800k rows)", path)
}
