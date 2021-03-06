package main

import (
	"bytes"
	"database/sql"
	"flag"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	droneS3BucketEnv := os.Getenv("DRONE_S3_BUCKET")
	if droneS3BucketEnv == "" {
		fmt.Print("missing DRONE_S3_BUCKET env\n")
		os.Exit(1)
	}

	droneDatabaseDatasource := os.Getenv("DRONE_DATABASE_DATASOURCE")
	if droneDatabaseDatasource == "" {
		fmt.Print("missing DRONE_DATABASE_DATASOURCE env\n")
		os.Exit(1)
	}

	fromIDFlag := flag.Int64("from-id", 0, "")
	toIDFlag := flag.Int64("to-id", math.MaxInt64, "")

	flag.Parse()

	fromID := *fromIDFlag
	toID := *toIDFlag

	db, err := sql.Open("mysql", droneDatabaseDatasource)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		panic(err)
	}

	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	awsSession, err := session.NewSession()
	if err != nil {
		panic(err)
	}

	awsS3Uploader := s3manager.NewUploader(awsSession)

	rows, err := db.Query(fmt.Sprintf("select log_id, log_data from logs where log_id >= %d and log_id <= %d", fromID, toID))
	if err != nil {
		panic(err)
	}

	for rows.Next() {
		var logID int64
		var logData []byte

		if err := rows.Scan(&logID, &logData); err != nil {
			panic(err)
		}

		if _, err = awsS3Uploader.Upload(&s3manager.UploadInput{
			Bucket: aws.String(droneS3BucketEnv),
			Key:    aws.String(fmt.Sprint(logID)),
			Body:   bytes.NewReader(logData),
		}); err != nil {
			panic(err)
		}

		fmt.Printf("%d\n", logID)
	}
}
