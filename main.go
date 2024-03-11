package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Body struct {
	Filters map[string]string `json:"filters"`
	Search  map[string]string `json:"search"`
}

type SessionData struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	UserId    string `json:"userId"`
	CreatedAt string `json:"createdAt"`
}

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	inputCsv, err := os.Open("accounts.csv")
	if err != nil {
		panic(err)
	}
	defer inputCsv.Close()

	accounts, err := csv.NewReader(inputCsv).ReadAll()
	if err != nil {
		panic(err)
	}

	outputCsv, err := os.Create("output.csv")
	if err != nil {
		panic(err)
	}
	defer outputCsv.Close()

	outputCsvWriter := csv.NewWriter(outputCsv)
	defer outputCsvWriter.Flush()

	for _, account := range accounts {
		params := url.Values{}
		params.Set("scope", "global")
		params.Set("offset", "0")
		params.Set("limit", "10")

		yotiUrl := "https://identity.yoti.com/api/applications/8ee36876-a433-4efe-8a60-119659a5aa39/idv/sessions/search?" + params.Encode()

		userId := fmt.Sprintf("%s_%s", strings.ToUpper(account[1]), account[2])
		body := Body{
			Filters: map[string]string{
				"status": "COMPLETED",
			},
			Search: map[string]string{
				"userId": userId,
			},
		}

		bodyJson, err := json.Marshal(body)
		if err != nil {
			panic(err)
		}

		req, err := http.NewRequest(http.MethodPost, yotiUrl, bytes.NewBuffer(bodyJson))
		if err != nil {
			panic(err)
		}

		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("authority", "identity.yoti.com")
		req.Header.Add("Cookie", "yoti-iam-gdpr-cookie=notification-accepted; _iam_csrf=MTcxMDE2OTAzNXxJa3hFU2xOWVJEQXJialJqYjI4eGNubGFNVGhIWjBoYVZUbGlkR2RQTWtaRWRVeEhObmt4V1ZwUk5HODlJZ289fBZMGRUN04JTTTNU-APRmfXSXDs95gZ-klSk2MYodMIU; token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MTAxODcwNzQsImp0aSI6IjQ3MmFmNTZhLTk0MWQtNDg0NS1hYzk1LTdmOTU5NTFkNTJiMCIsImlhdCI6MTcxMDE2OTA3NCwiaWQiOjE1NTk5LCJtZmFfY29tcGxldGUiOnRydWUsIm1mYV9tZXRob2QiOiJUT1RQIn0.Pz7QV-7VVFLFCt10fLwyF01vKSvOGkBzUpsQ6iF9Dr4")
		req.Header.Add("origin", "https://identity.yoti.com")
		req.Header.Add("referer", "https://identity.yoti.com/applications/8ee36876-a433-4efe-8a60-119659a5aa39/sessions-global?index=1&rows=20&details=APFJy%252F0TVl8mxDmOj2X684zfGW4jFdRjGT5ZaudOXNz3WAPhOO7A%252Fx91CFnBHedRO9NAop%252F5%252BYPkE%252FG2w%252BFqZh%252BAscbh1HORE4QVxQ1E%252F6hJk4vRx3j%252BR4%252FME3M3irkA1nHNQu%252Fg7eReY27ALtGvWwpl1W4wuPCdbdT9w%252F7J7z8%253D")
		req.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		resBody, err := io.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}

		var sessionDataList []SessionData
		if err = json.Unmarshal(resBody, &sessionDataList); err != nil {
			panic(err)
		}

		var sessions []string
		for _, session := range sessionDataList {
			sessions = append(sessions, session.ID)
		}

		var record []string
		record = append(record, userId)
		record = append(record, sessions...)
		if err := outputCsvWriter.Write(record); err != nil {
			panic(err)
		}

		switch len(sessionDataList) {
		case 0:
			log.Error().Str("userId", userId).Msg("no session found")

		case 1:
			log.Info().Str("userId", userId).Str("sessionId", sessions[0]).Msg("session found")

		default:

			log.Warn().Str("userId", userId).Strs("sessionIds", sessions).Msg("multiple sessions found")
		}

		time.Sleep(500 * time.Millisecond)
	}
}
