let seqTimes = [];
let ternTimes = [];
let sizes = [];
let chartInstance = null;

document.addEventListener("DOMContentLoaded", () => {
  document.getElementById("searchButton")?.addEventListener("click", searchData);
});

async function fetchOnce(size, code) {
  const response = await fetch("/search", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ size, code }),
  });

  if (!response.ok) {
    const text = await response.text().catch(() => "");
    throw new Error(`HTTP ${response.status} - ${text}`);
  }
  return await response.json();
}

function median(arr) {
  const a = [...arr].sort((x, y) => x - y);
  const m = Math.floor(a.length / 2);
  return a.length % 2 ? a[m] : (a[m - 1] + a[m]) / 2;
}

function pushOrUpdatePoint(n, seq, tern) {
  const idx = sizes.indexOf(n);
  if (idx === -1) {
    sizes.push(n);
    seqTimes.push(seq);
    ternTimes.push(tern);
  } else {
    const alpha = 0.35;
    seqTimes[idx] = seqTimes[idx] * (1 - alpha) + seq * alpha;
    ternTimes[idx] = ternTimes[idx] * (1 - alpha) + tern * alpha;
  }
}

async function searchData() {
  const btn = document.getElementById("searchButton");
  const size = parseInt((document.getElementById("size")?.value || "").trim(), 10);
  const code = (document.getElementById("code")?.value || "").trim().toUpperCase();

  if (!Number.isFinite(size) || size <= 0) {
    alert("Ukuran data (N) harus angka dan > 0. Contoh: 10000");
    return;
  }
  if (!/^B\d{6}$/.test(code)) {
    alert("Format kode harus seperti B000001");
    return;
  }

  btn.disabled = true;
  const old = btn.textContent;
  btn.textContent = "Memproses...";

  try {
    const repeats = 9; 
    const seqArr = [];
    const ternArr = [];
    let last = null;

    for (let i = 0; i < repeats; i++) {
      last = await fetchOnce(size, code);

      const s = Number(last?.seqTime);
      const t = Number(last?.ternTime);

      if (Number.isFinite(s) && Number.isFinite(t)) {
        seqArr.push(s);
        ternArr.push(t);
      }
    }

    if (!seqArr.length) {
      console.error("Response invalid:", last);
      alert("Data waktu tidak valid. Cek console (F12).");
      return;
    }

    const seq = median(seqArr);
    const tern = median(ternArr);
    const unit = (last?.unit || "µs").toString();

    renderResult(last);
    document.getElementById("seqTime").textContent =
      `Waktu Pencarian Sequential Iteratif: ${seq.toFixed(3)} ${unit}`;
    document.getElementById("ternTime").textContent =
      `Waktu Pencarian Sequential rekursif: ${tern.toFixed(3)} ${unit}`;

    pushOrUpdatePoint(size, seq, tern);
    renderChart(unit);
  } catch (e) {
    console.error(e);
    alert(
      "Gagal request.\nPastikan:\n" +
      "1) go run main.go sudah jalan\n" +
      "2) buka lewat http://localhost:8080 (bukan file://)\n"
    );
  } finally {
    btn.disabled = false;
    btn.textContent = old;
  }
}

function renderResult(data) {
  const out = document.getElementById("foundOut");
  if (!out) return;

  if (data?.found) {
    const f = data.found;
    out.textContent = `Ditemukan: ${f.name} (${f.code}) — Rp ${formatRupiah(f.price)}`;
  } else {
    out.textContent = "Barang tidak ditemukan pada ukuran N ini.";
  }
}

function formatRupiah(num) {
  const n = Math.round(Number(num) || 0);
  return n.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ".");
}

function renderChart(unit) {
  const canvas = document.getElementById("myChart");
  if (!canvas) return;
  const ctx = canvas.getContext("2d");

  if (chartInstance) chartInstance.destroy();

  chartInstance = new Chart(ctx, {
    type: "line",
    data: {
      labels: sizes,
      datasets: [
        { label: `Sequential (${unit})`, data: seqTimes, fill: true, tension: 0, borderWidth: 2, pointRadius: 4 },
        { label: `Sequential (${unit})`, data: ternTimes, fill: true, tension: 0, borderWidth: 2, pointRadius: 4 },
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: true,
      scales: {
        y: { beginAtZero: true, title: { display: true, text: `Waktu (${unit})` } },
        x: { title: { display: true, text: "Ukuran N (Jumlah Data)" } },
      },
      plugins: { legend: { display: true }, tooltip: { enabled: true } },
    },
  });
}
