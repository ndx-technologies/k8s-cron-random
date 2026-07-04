package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const doc = `k8s-cron-random - randomize Kubernetes CronJob schedules

Usage:

`

func main() {
	var (
		kubeconfig string
		masterURL  string
		namespace  string
		dryRun     bool
	)
	flag.StringVar(&kubeconfig, "kubeconfig", os.Getenv("KUBECONFIG"), "path to kubeconfig file, uses in-cluster config if empty")
	flag.StringVar(&masterURL, "master", "", "URL of the Kubernetes API server")
	flag.StringVar(&namespace, "namespace", "default", "Kubernetes namespace")
	flag.BoolVar(&dryRun, "dry-run", false, "do not apply changes")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), doc)
		flag.PrintDefaults()
	}
	flag.Parse()

	ctx := context.Background()

	config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		log.Fatal(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	cronJobs, err := clientset.BatchV1().CronJobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	changes := 0

	for i := range cronJobs.Items {
		cj := &cronJobs.Items[i]
		oldSchedule := cj.Spec.Schedule
		newSchedule := randomizeSchedule(oldSchedule, rng)

		if oldSchedule == newSchedule {
			fmt.Printf("%s: %q (no change)\n", cj.Name, oldSchedule)
			continue
		}

		fmt.Printf("%s: %q -> %q\n", cj.Name, oldSchedule, newSchedule)

		if !dryRun {
			cj.Spec.Schedule = newSchedule
			if _, err := clientset.BatchV1().CronJobs(namespace).Update(ctx, cj, metav1.UpdateOptions{}); err != nil {
				log.Printf("error updating cronjob %s: %s\n", cj.Name, err)
				continue
			}
		}

		changes++
	}

	if dryRun {
		fmt.Printf("dry-run: ")
	}
	fmt.Printf("updated %d/%d cronjobs in %s\n", changes, len(cronJobs.Items), namespace)
}

var cronShortcuts = map[string]string{
	"@yearly":   "0 0 1 1 *",
	"@annually": "0 0 1 1 *",
	"@monthly":  "0 0 1 * *",
	"@weekly":   "0 0 * * 0",
	"@daily":    "0 0 * * *",
	"@midnight": "0 0 * * *",
	"@hourly":   "0 * * * *",
}

// randomizeSchedule while preserving the period.
// Supports shortcuts: @yearly, @annually, @monthly, @weekly, @daily, @midnight, @hourly.
func randomizeSchedule(schedule string, rng *rand.Rand) string {
	if expanded, ok := cronShortcuts[schedule]; ok {
		schedule = expanded
	}

	fields := strings.Fields(schedule)
	if len(fields) != 5 {
		return schedule
	}

	result := [5]string{
		randomizeField(fields[0], 0, 59, rng), // minute
		randomizeField(fields[1], 0, 23, rng), // hour
		fields[2],                             // day of month (1-31) - keep as-is to avoid invalid dates
		fields[3],                             // month (1-12) - keep as-is
		fields[4],                             // day of week (0-6) - keep as-is
	}

	return strings.Join(result[:], " ")
}

// randomizeField randomizes a cron field if it's a specific number.
// Returns the field unchanged if it's a wildcard, range, list, or step.
func randomizeField(field string, min, max int, rng *rand.Rand) string {
	if strings.ContainsAny(field, "*/,-") {
		return field
	}

	v, err := strconv.Atoi(field)
	if err != nil {
		return field
	}

	if v < min || v > max {
		return field
	}

	nv := rng.Intn(max-min+1) + min
	return strconv.Itoa(nv)
}
