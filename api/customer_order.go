package api

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type CustomerOrder struct {
	ID                int     `db:"Id"`
	CustomerID        int     `db:"customer_id"`
	CustomerFirstName string  `db:"FirstName"`
	CustomerLastName  string  `db:"LastName"`
	Num_gallons_order int     `db:"num_gallons_order"`
	Date              string  `db:"date"`
	Date_created      string  `db:"date_created"`
	Total_price       float64 `db:"total_price"`
	Status            string  `db:"status"`
}

func Customer_OrderRoutes(r *gin.Engine, db *sqlx.DB) {
	r.GET("/api/get_order", func(ctx *gin.Context) {
		var orders []CustomerOrder
		query := `
			SELECT 
				co.Id, 
				co.customer_id, 
				a.FirstName, 
				a.LastName, 
				co.num_gallons_order, 
				co.date, 
				co.date_created,
				co.total_price,
				co.status
			FROM 
				customer_order co
			LEFT JOIN 
				Accounts a ON co.customer_id = a.Id
		`
		err := db.Select(&orders, query)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, orders)
	})

	r.POST("/api/save_order", func(ctx *gin.Context) {
		// Log incoming request
		log.Println("Received save order request")

		// Struct to bind JSON input
		var insertCustomerOrder struct {
			CustomerID        string `json:"customer_id"`
			Num_gallons_order string `json:"num_gallons_order"`
			Date              string `json:"date"`
			Status            string `json:"status"`
		}

		// Bind JSON and log any binding errors
		if err := ctx.ShouldBindJSON(&insertCustomerOrder); err != nil {
			log.Printf("JSON Binding Error: %v", err)
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid input",
				"details": err.Error(),
			})
			return
		}

		// Log received data for debugging
		log.Printf("Received Order Data: %+v", insertCustomerOrder)

		// Validate customer ID
		if insertCustomerOrder.CustomerID == "" {
			log.Println("Error: Customer ID is NULL")
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": "Customer ID is required",
			})
			return
		}

		// Convert num_gallons_order to int
		numGallons, err := strconv.Atoi(insertCustomerOrder.Num_gallons_order)
		if err != nil {
			log.Printf("Error converting num_gallons_order: %v", err)
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid number of gallons",
				"details": err.Error(),
			})
			return
		}

		// Validate date
		if insertCustomerOrder.Date == "" {
			insertCustomerOrder.Date = time.Now().Format("2006-01-02")
		}

		// Calculate total price based on inventory price
		var totalPrice float64
		getPriceQuery := `
			SELECT price * ? 
			FROM inventory_available 
			ORDER BY last_updated DESC 
			LIMIT 1
		`
		err = db.Get(&totalPrice, getPriceQuery, numGallons)
		if err != nil {
			log.Printf("Error calculating total price: %v", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to calculate price",
				"details": err.Error(),
			})
			return
		}

		// Default status if not provided
		if insertCustomerOrder.Status == "" {
			insertCustomerOrder.Status = "Pending"
		}

		// Start a transaction
		tx, err := db.Beginx()
		if err != nil {
			log.Printf("Error starting transaction: %v", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to start transaction",
				"details": err.Error(),
			})
			return
		}
		defer tx.Rollback() // Rollback in case of any error

		// Prepare insert query for customer order
		insertQuery := `
			INSERT INTO customer_order 
			(customer_id, num_gallons_order, date, date_created, total_price, status) 
			VALUES (?, ?, ?, NOW(), ?, ?)`

		// Execute the query
		result, err := tx.Exec(insertQuery,
			insertCustomerOrder.CustomerID,
			numGallons,
			insertCustomerOrder.Date,
			totalPrice,
			insertCustomerOrder.Status,
		)

		if err != nil {
			log.Printf("Database Insertion Error: %v", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to save order",
				"details": err.Error(),
			})
			return
		}

		// Get last inserted ID
		lastID, err := result.LastInsertId()
		if err != nil {
			log.Printf("Error getting last insert ID: %v", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to retrieve order ID",
				"details": err.Error(),
			})
			return
		}

		// If status is Pending, subtract from inventory
		if insertCustomerOrder.Status == "Pending" {
			updateInventoryQuery := `
				UPDATE inventory_available 
				SET total_quantity = total_quantity - ?, 
				    last_updated = NOW()
				WHERE inventory_id = (
					SELECT inventory_id 
					FROM inventory_available 
					ORDER BY last_updated DESC 
					LIMIT 1
				)
			`
			_, err = tx.Exec(updateInventoryQuery, numGallons)
			if err != nil {
				log.Printf("Error updating inventory: %v", err)
				ctx.JSON(http.StatusInternalServerError, gin.H{
					"error":   "Failed to update inventory",
					"details": err.Error(),
				})
				return
			}
		}

		// Commit the transaction
		err = tx.Commit()
		if err != nil {
			log.Printf("Error committing transaction: %v", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to save order and update inventory",
				"details": err.Error(),
			})
			return
		}

		// Successful response
		log.Printf("Order saved successfully. ID: %d", lastID)
		ctx.JSON(http.StatusOK, gin.H{
			"message":     "Order saved successfully",
			"order_id":    lastID,
			"total_price": totalPrice,
			"order":       insertCustomerOrder,
		})
	})
}
