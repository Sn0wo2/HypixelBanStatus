package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sync"
	"time"
)

type PunishmentStats struct {
	Success bool `json:"success"`
	Record  struct {
		WatchdogTotal int `json:"watchdog_total"`
		StaffTotal    int `json:"staff_total"`
	} `json:"record"`
	Timestamp time.Time
}

const (
	maxStatsCount = 24
	htmlTemplate  = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Hypixel Punishment Stats</title>
    <style>
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            background-color: #f0f2f5;
            margin: 0;
            padding: 0;
            display: flex;
            justify-content: center;
            align-items: center;
            min-height: 100vh;
        }
        .container {
            background-color: #ffffff;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            padding: 30px;
            max-width: 600px;
            width: 100%;
        }
        h1 {
            color: #1a237e;
            text-align: center;
            margin-bottom: 30px;
        }
        .stats-container {
            display: flex;
            justify-content: space-between;
            margin-bottom: 30px;
        }
        .stat-box {
            background-color: #e8eaf6;
            border-radius: 8px;
            padding: 20px;
            width: 45%;
        }
        .stat-title {
            font-size: 18px;
            font-weight: bold;
            color: #3f51b5;
            margin-bottom: 10px;
        }
        .stat-value {
            font-size: 24px;
            font-weight: bold;
            color: #1a237e;
        }
        .increase {
            color: #4caf50;
            font-weight: bold;
        }
        .last-updated {
            text-align: center;
            color: #757575;
            font-style: italic;
        }
        .disclaimer {
            text-align: center;
            color: #757575;
            font-size: 12px;
            margin-top: 20px;
        }
        .made-with-love {
            text-align: center;
            color: #757575;
            font-size: 14px;
            margin-top: 10px;
        }
        a {
            color: #3f51b5;
            text-decoration: none;
        }
        a:hover {
            text-decoration: underline;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Hypixel Punishment Stats</h1>
        <div class="stats-container">
            <div class="stat-box">
                <div class="stat-title">Watchdog Total</div>
                <div class="stat-value">{{.CurrentStats.Record.WatchdogTotal}}</div>
                <div>Last 5 minutes banned: <span class="increase">{{.WatchdogIncrease}}</span></div>
            </div>
            <div class="stat-box">
                <div class="stat-title">Staff Total</div>
                <div class="stat-value">{{.CurrentStats.Record.StaffTotal}}</div>
                <div>Last 5 minutes banned: <span class="increase">{{.StaffIncrease}}</span></div>
            </div>
        </div>
        <p class="last-updated">Last updated: {{.LastUpdated}}</p>
        <p class="disclaimer">This website is not affiliated with <a href="https://hypixel.net/" target="_blank">Hypixel Inc</a> or <a href="https://www.minecraft.net/" target="_blank">Mojang</a> in any way.</p>
        <p class="made-with-love">Made with ❤</p>
    </div>
    <script>
        function animateValue(obj, start, end, duration) {
            let startTimestamp = null;
            const step = (timestamp) => {
                if (!startTimestamp) startTimestamp = timestamp;
                const progress = Math.min((timestamp - startTimestamp) / duration, 1);
                obj.innerHTML = Math.floor(progress * (end - start) + start);
                if (progress < 1) {
                    window.requestAnimationFrame(step);
                }
            };
            window.requestAnimationFrame(step);
        }

        document.addEventListener('DOMContentLoaded', (event) => {
            const watchdogTotal = document.querySelector('.stat-box:nth-child(1) .stat-value');
            const staffTotal = document.querySelector('.stat-box:nth-child(2) .stat-value');
            
            animateValue(watchdogTotal, 0, parseInt(watchdogTotal.innerHTML), 1000);
            animateValue(staffTotal, 0, parseInt(staffTotal.innerHTML), 1000);
        });
    </script>
</body>
</html>
`
)

var (
	statsList []PunishmentStats
	mutex     sync.Mutex
)

func fetchStats() (*PunishmentStats, error) {
	resp, err := http.Get("https://api.plancke.io/hypixel/v1/punishmentStats")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var stats PunishmentStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, err
	}
	if !stats.Success {
		return nil, fmt.Errorf("API request was not successful")
	}
	stats.Timestamp = time.Now()
	return &stats, nil
}

func updateStats() {
	for {
		if newStats, err := fetchStats(); err != nil {
			fmt.Println("Error fetching stats:", err)
		} else {
			mutex.Lock()
			statsList = append(statsList, *newStats)
			if len(statsList) > maxStatsCount {
				statsList = statsList[1:]
			}
			mutex.Unlock()
		}
		time.Sleep(15 * time.Second)
	}
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.New("stats").Parse(htmlTemplate))
	mutex.Lock()
	defer mutex.Unlock()
	if len(statsList) == 0 {
		http.Error(w, "No stats available", http.StatusInternalServerError)
		return
	}
	currentStats := statsList[len(statsList)-1]
	fiveMinutesAgo := time.Now().Add(-5 * time.Minute)
	var previousStats PunishmentStats
	for i := len(statsList) - 1; i >= 0; i-- {
		if statsList[i].Timestamp.Before(fiveMinutesAgo) {
			previousStats = statsList[i]
			break
		}
	}
	var watchdogIncrease, staffIncrease interface{} = "N/A", "N/A"
	if !previousStats.Timestamp.IsZero() {
		watchdogIncrease = currentStats.Record.WatchdogTotal - previousStats.Record.WatchdogTotal
		staffIncrease = currentStats.Record.StaffTotal - previousStats.Record.StaffTotal
	}
	data := struct {
		CurrentStats     PunishmentStats
		WatchdogIncrease interface{}
		StaffIncrease    interface{}
		LastUpdated      string
	}{currentStats, watchdogIncrease, staffIncrease, currentStats.Timestamp.Format(time.RFC1123)}
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func main() {
	go updateStats()
	http.HandleFunc("/", handleRoot)
	fmt.Println("Server is running on http://localhost:80")
	if err := http.ListenAndServe(":80", nil); err != nil {
		fmt.Println("Error starting server:", err)
	}
}
