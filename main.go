package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
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
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
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
            max-width: 800px;
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
            cursor: pointer;
            transition: background-color 0.3s;
        }
        .stat-box:hover {
            background-color: #c5cae9;
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
        .chart-container {
            margin-top: 30px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Hypixel Punishment Stats</h1>
        <div class="stats-container">
            <div class="stat-box" onclick="location.href='/watchdog';">
                <div class="stat-title">Watchdog Total</div>
                <div class="stat-value">{{.CurrentStats.Record.WatchdogTotal}}</div>
                <div>Last 5 minutes banned: <span class="increase">{{.WatchdogIncrease}}</span></div>
            </div>
            <div class="stat-box" onclick="location.href='/staff';">
                <div class="stat-title">Staff Total</div>
                <div class="stat-value">{{.CurrentStats.Record.StaffTotal}}</div>
                <div>Last 5 minutes banned: <span class="increase">{{.StaffIncrease}}</span></div>
            </div>
        </div>
        <div class="chart-container">
            <canvas id="banChart"></canvas>
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

    const ctx = document.getElementById('banChart').getContext('2d');

    const watchdogDiff = {{.WatchdogData}}.map((v, i, arr) => i === 0 ? 0 : v - arr[i-1]);
    const staffDiff = {{.StaffData}}.map((v, i, arr) => i === 0 ? 0 : v - arr[i-1]);

    new Chart(ctx, {
        type: 'line',
        data: {
            labels: {{.ChartLabels}},
            datasets: [{
                label: 'Watchdog Bans (Total)',
                data: {{.WatchdogData}},
                borderColor: 'rgb(75, 192, 192)',
                yAxisID: 'y-axis-1',
            }, {
                label: 'Staff Bans (Total)',
                data: {{.StaffData}},
                borderColor: 'rgb(255, 99, 132)',
                yAxisID: 'y-axis-1',
            }, {
                label: 'Watchdog Bans (Change)',
                data: watchdogDiff,
                borderColor: 'rgb(75, 192, 192)',
                borderDash: [5, 5],
                yAxisID: 'y-axis-2',
            }, {
                label: 'Staff Bans (Change)',
                data: staffDiff,
                borderColor: 'rgb(255, 99, 132)',
                borderDash: [5, 5],
                yAxisID: 'y-axis-2',
            }]
        },
        options: {
            responsive: true,
            scales: {
                'y-axis-1': {
                    type: 'linear',
                    display: true,
                    position: 'left',
                    ticks: {
                        callback: function(value, index, values) {
                            return value.toString().slice(-5);  // 只显示最后5位数字
                        }
                    }
                },
                'y-axis-2': {
                    type: 'linear',
                    display: true,
                    position: 'right',
                    ticks: {
                        max: 5,
                        min: -1
                    }
                }
            }
        }
    });
});
    </script>
</body>
</html>
`
	watchdogTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Watchdog Stats</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
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
            max-width: 800px;
            width: 100%;
        }
        h1 {
            color: #1a237e;
            text-align: center;
            margin-bottom: 30px;
        }
        .chart-container {
            margin-top: 30px;
            height: 400px;
        }
        .last-updated {
            text-align: center;
            color: #757575;
            font-style: italic;
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
        <h1>Watchdog Stats</h1>
        <div class="chart-container">
            <canvas id="watchdogChart"></canvas>
        </div>
        <p class="last-updated">Last updated: {{.LastUpdated}}</p>
        <p><a href="/">Back to main page</a></p>
    </div>
    <script>
        const ctx = document.getElementById('watchdogChart').getContext('2d');
        const watchdogChange = {{.Data}}.map((v, i, arr) => i === 0 ? 0 : v - arr[i-1]);
        new Chart(ctx, {
            type: 'line',
            data: {
                labels: {{.Labels}},
                datasets: [{
                    label: 'Watchdog Bans (Total)',
                    data: {{.Data}},
                    borderColor: 'rgb(75, 192, 192)',
                    yAxisID: 'y-axis-1',
                }, {
                    label: 'Watchdog Bans (Change)',
                    data: watchdogChange,
                    borderColor: 'rgb(75, 192, 192)',
                    borderDash: [5, 5],
                    yAxisID: 'y-axis-2',
                }]
            },
            options: {
                responsive: true,
                scales: {
                    'y-axis-1': {
                        type: 'linear',
                        display: true,
                        position: 'left',
                        ticks: {
                            callback: function(value, index, values) {
                                return value.toString().slice(-5);
                            }
                        }
                    },
                    'y-axis-2': {
                        type: 'linear',
                        display: true,
                        position: 'right',
                        ticks: {
                            max: 5,
                            min: -1
                        }
                    }
                }
            }
        });
    </script>
</body>
</html>
`

	staffTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Staff Stats</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
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
            max-width: 800px;
            width: 100%;
        }
        h1 {
            color: #1a237e;
            text-align: center;
            margin-bottom: 30px;
        }
        .chart-container {
            margin-top: 30px;
            height: 400px;
        }
        .last-updated {
            text-align: center;
            color: #757575;
            font-style: italic;
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
        <h1>Staff Stats</h1>
        <div class="chart-container">
            <canvas id="staffChart"></canvas>
        </div>
        <p class="last-updated">Last updated: {{.LastUpdated}}</p>
        <p><a href="/">Back to main page</a></p>
    </div>
    <script>
        const ctx = document.getElementById('staffChart').getContext('2d');
        const staffChange = {{.Data}}.map((v, i, arr) => i === 0 ? 0 : v - arr[i-1]);
        new Chart(ctx, {
            type: 'line',
            data: {
                labels: {{.Labels}},
                datasets: [{
                    label: 'Staff Bans (Total)',
                    data: {{.Data}},
                    borderColor: 'rgb(255, 99, 132)',
                    yAxisID: 'y-axis-1',
                }, {
                    label: 'Staff Bans (Change)',
                    data: staffChange,
                    borderColor: 'rgb(255, 99, 132)',
                    borderDash: [5, 5],
                    yAxisID: 'y-axis-2',
                }]
            },
            options: {
                responsive: true,
                scales: {
                    'y-axis-1': {
                        type: 'linear',
                        display: true,
                        position: 'left',
                        ticks: {
                            callback: function(value, index, values) {
                                return value.toString().slice(-5);
                            }
                        }
                    },
                    'y-axis-2': {
                        type: 'linear',
                        display: true,
                        position: 'right',
                        ticks: {
                            max: 5,
                            min: -1
                        }
                    }
                }
            }
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
	// Please use self api endpoints
	resp, err := http.Get("Please use self api endpoints")
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println("Error closing response body:", err)
		}
	}(resp.Body)
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

	var chartLabels []string
	var watchdogData, staffData []int
	for i := len(statsList) - 1; i >= 0 && len(chartLabels) < 20; i-- {
		chartLabels = append([]string{statsList[i].Timestamp.Format("15:04:05")}, chartLabels...)
		watchdogData = append([]int{statsList[i].Record.WatchdogTotal}, watchdogData...)
		staffData = append([]int{statsList[i].Record.StaffTotal}, staffData...)
	}

	data := struct {
		CurrentStats     PunishmentStats
		WatchdogIncrease interface{}
		StaffIncrease    interface{}
		LastUpdated      string
		ChartLabels      []string
		WatchdogData     []int
		StaffData        []int
	}{
		currentStats,
		watchdogIncrease,
		staffIncrease,
		currentStats.Timestamp.Format(time.RFC1123),
		chartLabels,
		watchdogData,
		staffData,
	}
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleWatchdog(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.New("watchdog").Parse(watchdogTemplate))
	mutex.Lock()
	defer mutex.Unlock()

	var labels []string
	var data []int
	for i := len(statsList) - 1; i >= 0 && len(labels) < 24; i-- {
		labels = append([]string{statsList[i].Timestamp.Format("15:04:05")}, labels...)
		data = append([]int{statsList[i].Record.WatchdogTotal}, data...)
	}

	pageData := struct {
		Labels      []string
		Data        []int
		LastUpdated string
	}{
		Labels:      labels,
		Data:        data,
		LastUpdated: statsList[len(statsList)-1].Timestamp.Format(time.RFC1123),
	}

	if err := tmpl.Execute(w, pageData); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleStaff(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.New("staff").Parse(staffTemplate))
	mutex.Lock()
	defer mutex.Unlock()

	var labels []string
	var data []int
	for i := len(statsList) - 1; i >= 0 && len(labels) < 24; i-- {
		labels = append([]string{statsList[i].Timestamp.Format("15:04:05")}, labels...)
		data = append([]int{statsList[i].Record.StaffTotal}, data...)
	}

	pageData := struct {
		Labels      []string
		Data        []int
		LastUpdated string
	}{
		Labels:      labels,
		Data:        data,
		LastUpdated: statsList[len(statsList)-1].Timestamp.Format(time.RFC1123),
	}

	if err := tmpl.Execute(w, pageData); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func main() {
	go updateStats()
	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/watchdog", handleWatchdog)
	http.HandleFunc("/staff", handleStaff)
	fmt.Println("Server is running on http://localhost:80")
	if err := http.ListenAndServe(":80", nil); err != nil {
		fmt.Println("Error starting server:", err)
	}
}
