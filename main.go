package main

import (
	"bufio"
	"encoding/base64"
	"flag"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type metrics struct {
	txPower prometheus.Gauge
	rxPower prometheus.Gauge
}

func NewMetrics(reg prometheus.Registerer) *metrics {
	m := &metrics{
		txPower: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "pon_tx_power",
			Help: "PON transmit power in dBm",
		}),
		rxPower: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "pon_rx_power",
			Help: "PON receive power in dBm",
		}),
	}
	reg.MustRegister(m.rxPower)
	reg.MustRegister(m.txPower)
	return m
}

var httpClient = http.Client{Timeout: time.Second * 5}

func lookForDecibelMilliwatts(scanner *bufio.Scanner, lookup string) float64 {
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), lookup) {
			scanner.Scan() // next line contains the tx power
			// 14 is <td width=60%>
			// We also partition by space to get the number
			segments := strings.Split(strings.TrimSpace(scanner.Text())[14:], " ")
			if len(segments) == 1 {
				return 0 // Something bad happened?
			} else {
				tx, _ := strconv.ParseFloat(segments[0], 64)
				return tx // fuck the error I guess
			}
		}
	}
	return 0 // shash
}

func updateMetrics(m *metrics, adminPassword string) {
	// Login to router
	loginForm := url.Values{}
	loginForm.Add("challenge", "")
	loginForm.Add("username", "admin")
	loginForm.Add("save", "Login")
	loginForm.Add("encodePassword", adminPassword)
	loginForm.Add("submit-url", "/admin/login.asp")
	resp, err := httpClient.PostForm("http://192.168.254.1/boaform/admin/formLogin", loginForm)
	if err != nil {
		log.Println("cannot send login request:", err)
		return
	}
	resp.Body.Close() // we dont need the body
	if resp.StatusCode/100 == 4 || resp.StatusCode/100 == 5 {
		log.Panicln("login status code is not 2xx or 3xx:", resp.Status)
		return
	}
	// Request the PON status
	resp, err = httpClient.Get("http://192.168.254.1/status_pon.asp")
	if err != nil {
		log.Println("cannot send pon status request:", err)
		return
	}
	defer resp.Body.Close()
	// Read line by line
	scanner := bufio.NewScanner(resp.Body)
	m.txPower.Set(lookForDecibelMilliwatts(scanner, "Tx Power"))
	m.rxPower.Set(lookForDecibelMilliwatts(scanner, "Rx Power"))
}

func main() {
	// Command line arguments
	addr := flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")
	updateInterval := flag.Int("update-interval", 15, "Update interval in seconds.")
	adminPassword := flag.String("admin-password", "admin410", "The admin password of dashboard.")
	flag.Parse()

	// Create a registery for prometheus
	reg := prometheus.NewRegistry()

	// Create new metrics and register them using the custom registry.
	m := NewMetrics(reg)
	// Regularly update the values
	go func() {
		adminPassword := base64.StdEncoding.EncodeToString([]byte(*adminPassword))
		for {
			updateMetrics(m, adminPassword)
			time.Sleep(time.Second * time.Duration(*updateInterval))
		}
	}()

	// Expose metrics and custom registry via an HTTP server
	// using the HandleFor function. "/metrics" is the usual endpoint for that.
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	log.Fatal(http.ListenAndServe(*addr, nil))
}
