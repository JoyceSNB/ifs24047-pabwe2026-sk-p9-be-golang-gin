# PABWE 2026 Praktikum 9 — Delcom Blog Backend (Golang Gin)

Backend RESTful API untuk aplikasi **Delcom Blog** dengan arsitektur **Feature-First Approach & TDD (100% Code Coverage)**, dibangun menggunakan **Go**, **Gin Web Framework**, basis data **PostgreSQL (`pabwe_p9_blog`)**, dokumentasi interaktif **Swagger UI**, serta **AI Blog Generator** yang terintegrasi dengan Search Engine dan OpenAI Chat Completions.

---

## 🚀 Fitur Utama Backend

1. **Feature-First Architecture**:
   - Struktur modul dikelompokkan berdasarkan domain bisnis: `auth`, `post`, `ai`.
   - Menerapkan Dependency Inversion dengan repository interface dan service interface.
   - Pemisahan tanggung jawab yang jelas antara Handler, Service, Repository, DTO, dan Middleware.

2. **Artikel Blog (`Post`)**:
   - CRUD artikel milik penulis (`/api/posts`) dengan slug unik otomatis dan estimasi waktu baca (jumlah kata / 200 wpm).
   - Kategori tetap: Teknologi, Pendidikan, Gaya Hidup, Bisnis, Opini, Umum.
   - Toggle Draf ↔ Terbit secara instan (`PATCH /api/posts/:id/toggle-publish`) beserta `published_at`.
   - Statistik 3 kartu penulis: Total Artikel, Total Terbit, Belum Selesai (`GET /api/posts/stats`).

3. **Akses Publik tanpa Login**:
   - Daftar artikel terbit dengan pencarian `?q=` dan filter `?category=` (`GET /api/public/posts`).
   - Detail artikel berdasarkan slug beserta data penulis (`GET /api/public/posts/:slug`).
   - Daftar kategori (`GET /api/public/categories`).

4. **Autentikasi & Autorisasi (JWT)**:
   - Login menggunakan username atau email.
   - Role-Based Access Control: `admin` (monitoring seluruh pengguna dan artikel) dan `member` (penulis).
   - Endpoint admin: `/api/admin/users`, `/api/admin/posts`, `/api/admin/users/:id/posts`.

5. **AI Blog Generator**:
   - `POST /api/posts/ai-generate`: topik → penelusuran Search Engine → OpenAI Chat Completions → draf artikel utuh (judul, kategori, excerpt, konten Markdown, dan daftar sumber rujukan).
   - Konfigurasi via `.env`: `OPENAI_ENDPOINT`, `OPENAI_KEY`, `OPENAI_MODEL`, `SEARCH_ENGINE_ENDPOINT`.

6. **Dokumentasi OpenAPI / Swagger UI**:
   - Tersedia di `/swagger/index.html`; spesifikasi `api.json` di-embed ke binary (`/swagger/api.json` atau `/api.json`).

7. **Format Standar Respons API**:
   ```json
   {
     "status": "success|fail|error",
     "message": "Pesan deskripsi",
     "data": { ... }
   }
   ```
   Field `"data"` bersifat opsional.

8. **100% Test Coverage**:
   - Seluruh package `internal/config`, `internal/database`, `internal/middleware`, `internal/response`, `internal/features/...`, dan `internal/server` diuji dengan target **100.0% Statement Coverage**.

---

## ⚙️ Konfigurasi Environment (`.env`)

```env
APP_PORT=8081
APP_ENV=development
APP_KEY=pabwe-secret-key-2026-blog-golang-gin

# Default Admin Credentials
ADMIN_USERNAME=admin
ADMIN_PASSWORD=admin

# Database Configuration (Laravel-Style)
DB_CONNECTION=pgsql
DB_HOST=127.0.0.1
DB_PORT=5432
DB_DATABASE=pabwe_p9_blog
DB_USERNAME=postgres
DB_PASSWORD=postgres
DB_SSLMODE=disable

# AI / OpenAI Configuration
OPENAI_ENDPOINT=https://delcom.org/api/v1
OPENAI_KEY=key
OPENAI_MODEL=nama-model

# Search Engine (Wikipedia OpenSearch / DuckDuckGo, tanpa API key)
SEARCH_ENGINE_ENDPOINT=https://id.wikipedia.org/w/api.php
```

---

## 🧪 Menjalankan Pengujian (TDD 100% Coverage)

```bash
# Menjalankan seluruh pengujian unit & integrasi dengan laporan coverage:
go test -v -cover ./internal/...
```

---

## 🏃 Menjalankan Server (Mode Development)

```bash
# Menjalankan langsung melalui Go compiler:
go run main.go
```
- API Server aktif pada: `http://localhost:8081`
- Dokumentasi Swagger UI: `http://localhost:8081/swagger/index.html` (Spesifikasi JSON: `/swagger/api.json` atau `/api.json`)

---

## 📦 Mengompilasi (*Build*) & Menjalankan Hasil Kompilasi

Kompilasi kode sumber menjadi berkas binary mandiri (*standalone executable*) untuk keperluan produksi atau *deployment*:

### 1. Kompilasi (*Build*)
```bash
# Kompilasi untuk sistem operasi lokal (Windows):
go build -o server.exe main.go
```

> **Opsi Cross-Compile (misal untuk Server Linux/VPS):**
> ```bash
> # Di Windows PowerShell:
> $env:GOOS="linux"; $env:GOARCH="amd64"; go build -o server main.go
>
> # Di Windows Command Prompt (CMD):
> set GOOS=linux
> set GOARCH=amd64
> go build -o server main.go
> ```

### 2. Menjalankan Hasil *Build*
```bash
# Jalankan biner yang sudah dikompilasi (Windows):
.\server.exe

# Jika di Linux:
./server
```

> **Catatan Penting:**
> Pastikan berkas `.env` berada di direktori yang sama dengan berkas `server.exe` saat dijalankan agar konfigurasi basis data dan port aplikasi terbaca dengan benar.
