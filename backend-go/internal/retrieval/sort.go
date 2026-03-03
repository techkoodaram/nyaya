package retrieval

import "sort"

func SortResults(results []Result) {
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].Chunk.Title < results[j].Chunk.Title
		}
		return results[i].Score > results[j].Score
	})
}
