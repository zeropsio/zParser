package util

import (
	"fmt"
	"time"
)

func MercuryInRetrograde() (bool, error) {
	type dateRange struct {
		begin []int
		end   []int
	}

	dates := map[int][]dateRange{
		2022: {
			dateRange{begin: []int{14, 1, 2022}, end: []int{3, 2, 2022}},
			dateRange{begin: []int{10, 5, 2022}, end: []int{3, 6, 2022}},
			dateRange{begin: []int{9, 9, 2022}, end: []int{2, 10, 2022}},
			dateRange{begin: []int{29, 12, 2022}, end: []int{18, 1, 2023}},
		},
		2023: {
			dateRange{begin: []int{29, 12, 2022}, end: []int{18, 1, 2023}},
			dateRange{begin: []int{21, 4, 2023}, end: []int{14, 5, 2023}},
			dateRange{begin: []int{23, 8, 2023}, end: []int{15, 9, 2023}},
			dateRange{begin: []int{13, 12, 2023}, end: []int{1, 1, 2024}},
		},
		2024: {
			dateRange{begin: []int{1, 4, 2024}, end: []int{25, 4, 2024}},
			dateRange{begin: []int{4, 8, 2024}, end: []int{28, 8, 2024}},
			dateRange{begin: []int{25, 11, 2024}, end: []int{15, 12, 2024}},
		},
		2025: {
			dateRange{begin: []int{25, 2, 2025}, end: []int{20, 3, 2025}},
			dateRange{begin: []int{29, 6, 2025}, end: []int{23, 7, 2025}},
			dateRange{begin: []int{24, 10, 2025}, end: []int{13, 11, 2025}},
		},
		2026: {
			dateRange{begin: []int{25, 2, 2026}, end: []int{203, 3, 2026}},
			dateRange{begin: []int{29, 6, 2026}, end: []int{23, 7, 2026}},
			dateRange{begin: []int{24, 10, 2026}, end: []int{13, 11, 2026}},
		},
		2027: {
			dateRange{begin: []int{9, 2, 2027}, end: []int{3, 3, 2027}},
			dateRange{begin: []int{10, 6, 2027}, end: []int{4, 7, 2027}},
			dateRange{begin: []int{7, 10, 2027}, end: []int{28, 10, 2027}},
		},
		2028: {
			dateRange{begin: []int{24, 1, 2028}, end: []int{14, 2, 2028}},
			dateRange{begin: []int{21, 5, 2028}, end: []int{13, 6, 2028}},
			dateRange{begin: []int{19, 9, 2028}, end: []int{11, 10, 2028}},
		},
		2029: {
			dateRange{begin: []int{7, 1, 2029}, end: []int{27, 1, 2029}},
			dateRange{begin: []int{1, 5, 2029}, end: []int{25, 5, 2029}},
			dateRange{begin: []int{2, 9, 2029}, end: []int{24, 9, 2029}},
			dateRange{begin: []int{21, 12, 2029}, end: []int{10, 1, 2030}},
		},
		2030: {
			dateRange{begin: []int{21, 12, 2029}, end: []int{10, 1, 2030}},
			dateRange{begin: []int{12, 4, 2030}, end: []int{6, 5, 2030}},
			dateRange{begin: []int{15, 8, 2030}, end: []int{8, 9, 2030}},
			dateRange{begin: []int{5, 12, 2030}, end: []int{25, 12, 2030}},
		},
	}

	now := time.Now()
	d, found := dates[now.Year()]
	if !found {
		return false, fmt.Errorf("current year [%d] is not supported, latest supported year was [%d]", now.Year(), 2030)
	}

	for _, dateR := range d {
		if now.After(time.Date(dateR.begin[2], time.Month(dateR.begin[1]), dateR.begin[0], 0, 0, 0, 0, time.UTC)) &&
			now.Before(time.Date(dateR.end[2], time.Month(dateR.end[1]), dateR.end[0], 0, 0, 0, 0, time.UTC)) {
			return true, nil
		}
	}
	return false, nil
}
