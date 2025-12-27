package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"time"
)

type Item struct {
	Code  string  `json:"code"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

type SearchRequest struct {
	Size int    `json:"size"`
	Code string `json:"code"`
}

type SearchResponse struct {
	Found      *Item   `json:"found"`
	SeqTimeUs  float64 `json:"seqTime"`  // µs
	TernTimeUs float64 `json:"ternTime"` // µs
	Unit       string  `json:"unit"`
	Error      string  `json:"error,omitempty"`
}

var (
	masterItems []Item
	codeIndex   map[string]int
)

func makeCode(i int) string {
	//I.S. Diberikan variable i betipe integer dan bernilai ≥ 0.
	//F.S. Menghasilkan sebuah string kode barang dengan format BXXXXXX,
	//dimana XXXXXX adalah angka berjumlah 6 digit yang merepresentasikan i + 1.
	return fmt.Sprintf("B%06d", i+1) // B000001
}

func buildDataset(maxSize int) {
	//I.S. Variable global masterItems dan codeIndex belum terisi data barang.
	//F.s. variable masterItems berisi data barang maxSize elemen,
	// dan codeIndex berisi pasangan kode barang dengan indeksnya pada masterItems.
	rng := rand.New(rand.NewSource(42))

	names := []string{
		"Beras", "Gula", "Minyak Goreng", "Mie Instan", "Kopi", "Teh",
		"Susu", "Roti", "Telur", "Garam", "Sabun", "Shampoo",
		"Pasta Gigi", "Tisu", "Deterjen", "Air Mineral", "Sarden",
		"Kecap", "Saos", "Biskuit", "Cokelat", "Chips", "Saus Sambal",
		"Masker", "Hand Sanitizer", "Baterai", "Lampu", "Pulpen", "Buku Tulis",
	}

	masterItems = make([]Item, maxSize)
	codeIndex = make(map[string]int, maxSize)

	for i := 0; i < maxSize; i++ {
		code := makeCode(i)
		name := names[rng.Intn(len(names))]
		price := float64(2000 + rng.Intn(248000)) // 2.000 - 250.000

		masterItems[i] = Item{
			Code:  code,
			Name:  name,
			Price: price,
		}
		codeIndex[code] = i
	}
}

func sequentialSearchByCodeIterative(data []Item, target string) int {
	//I.S. Array data berisi kumpulan barang dan target adalah kode yang dicari.
	//F.S. Mengembalikan indeks data jika ditemukan dan -1 jika tidak ditemukan.
	for i := 0; i < len(data); i++ {
		if data[i].Code == target {
			return i
		}

	}
	return -1
}

func sequentialSearchByCodeRecursive(data []Item, target string, index int) int {
	if index >= len(data) {
		return -1
	}
	if data[index].Code == target {
		return index
	}
	return sequentialSearchByCodeRecursive(data, target, index+1)
}

func measureTime(data []Item, targetCode string) (seqUs float64, ternUs float64, found *Item) {
	//I.S. Data barang tersedia dan kode target sudah ditentukan.
	//F.S. Menghasilkan waktu eksekusi sequential search iteratif dan rekursif serta data barang jika diteentukan.
	sortedByCode := make([]Item, len(data))
	copy(sortedByCode, data)
	sort.Slice(sortedByCode, func(i, j int) bool { return sortedByCode[i].Code < sortedByCode[j].Code })

	const reps = 200000

	t0 := time.Now()
	seqIdx := -1
	for i := 0; i < reps; i++ {
		seqIdx = sequentialSearchByCodeIterative(data, targetCode)
	}
	seqUs = float64(time.Since(t0).Nanoseconds()) / 1e3 / float64(reps)

	t1 := time.Now()
	ternIdx := -1
	for i := 0; i < reps; i++ {
		ternIdx = sequentialSearchByCodeRecursive(sortedByCode, targetCode, 0)
	}
	ternUs = float64(time.Since(t1).Nanoseconds()) / 1e3 / float64(reps)

	if seqIdx != -1 {
		tmp := data[seqIdx]
		found = &tmp
	} else if ternIdx != -1 {
		tmp := sortedByCode[ternIdx]
		found = &tmp
	} else {
		found = nil
	}
	return
}

func withCORS(next http.Handler) http.Handler {
	//I.S. Handler HTTP tersedia tanpa konfigurasi CORS.
	//F.S. Menghasilkan handler baru yang mendukung permintaan CORS.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func respondJSON(w http.ResponseWriter, status int, v any) {
	//I.S. Response HTTP belum dikirim ke client.
	//F.S. Response JSON dikirim ke client dengan status HTTP yang sesuai.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	//I.S. Client mengirim request POST berisi ukuran data dan kode barang.
	//F.S. Server mengirim hasil pencarian dan waktu eksekusi algoritma dalam format JSON.
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, SearchResponse{Error: "Invalid JSON body"})
		return
	}

	req.Code = strings.ToUpper(strings.TrimSpace(req.Code))

	if req.Size <= 0 {
		respondJSON(w, http.StatusBadRequest, SearchResponse{Error: "Size harus > 0"})
		return
	}
	if req.Size > len(masterItems) {
		respondJSON(w, http.StatusBadRequest, SearchResponse{Error: fmt.Sprintf("Size terlalu besar. Maksimum %d", len(masterItems))})
		return
	}
	if !isValidCode(req.Code) {
		respondJSON(w, http.StatusBadRequest, SearchResponse{Error: "Format kode harus seperti B000001"})
		return
	}

	subset := masterItems[:req.Size]

	if idx, ok := codeIndex[req.Code]; ok && idx >= req.Size {
		respondJSON(w, http.StatusOK, SearchResponse{
			Found:      nil,
			SeqTimeUs:  0,
			TernTimeUs: 0,
			Unit:       "µs",
		})
		return
	}

	seqUs, ternUs, found := measureTime(subset, req.Code)

	respondJSON(w, http.StatusOK, SearchResponse{
		Found:      found,
		SeqTimeUs:  seqUs,
		TernTimeUs: ternUs,
		Unit:       "µs",
	})
}

func isValidCode(code string) bool {
	//I.S. Input berupa string kode barang.
	//F.S. Mengembalikan nilai true jika format kode sesuai BXXXXXX, false jika tidak.
	if len(code) != 7 || code[0] != 'B' {
		return false
	}
	for i := 1; i < 7; i++ {
		if code[i] < '0' || code[i] > '9' {
			return false
		}
	}
	return true
}

func main() {
	//I.S. Program belum berjalan dan server belum aktif.
	//F.S. Dataset dibangkitkan dan server HTTP berjalan pada port 8080.
	buildDataset(200000)
	mux := http.NewServeMux()
	mux.HandleFunc("/search", searchHandler)
	mux.Handle("/", http.FileServer(http.Dir(".")))
	port := 8080
	log.Printf("Server running on http://localhost:%d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), withCORS(mux)))
}
