# Catalog development

Panduan ini memakai modul Brand sebagai contoh. Mulai saat Catalog sudah bisa dijalankan, terhubung ke PostgreSQL, dan punya route admin dengan `RequireAdmin`.

## Urutan pengerjaan Brand

1. Buat [migration](migrations/) untuk tabel `brands`. Di repo ini `000001` membuat tabel, `000002` membatasi panjang nama, dan `000003` menambah status. Siapkan file `.up.sql` dan `.down.sql` untuk tiap perubahan.
2. Buat [model](internal/model/brand.go) untuk mewakili satu baris database. Model Brand memuat ID, nama, key gambar opsional, status, dan timestamp.
3. Buat [repository](internal/repository/brand.go) dengan operasi `Create`, `List`, dan `Update`. Repository menangani query PostgreSQL, termasuk search, pagination, dan konflik nama unik.
4. Buat [DTO](internal/dto/brand.go) untuk JSON request dan response. Pisahkan request create dan update supaya aturan field tiap operasi mudah dilihat.
5. Buat [service](internal/service/brand.go). Trim dan validasi nama, cek gambar di R2 saat ada key baru, panggil repository, lalu bentuk DTO response.
6. Buat [handler](internal/handler/brand.go) untuk membaca request dan mengirim HTTP response. Daftarkan endpoint di [router](internal/router/router.go) pada group `/admin`.
7. Hubungkan repository, service, dan handler di [main.go](main.go), lalu berikan handler ke router.

Urutan membaca satu request adalah router → handler → service → repository. PostgreSQL mengembalikan data ke service, lalu handler mengirim hasilnya sebagai JSON.

## Coba Brand di Bruno

Jalankan Identity di port `8081`. Dari folder Catalog, jalankan migration dengan `DATABASE_URL` yang sudah tersedia di terminal, lalu jalankan Catalog. Contoh ini memakai port default Catalog `8082`.

```sh
migrate -path migrations -database "$DATABASE_URL" up
go run .
```

1. Kirim `POST http://localhost:8081/api/v1/identity/login` dengan email dan password OWNER atau ADMIN. Salin `data.token` dan masukkan ke **Auth → Bearer Token** untuk request Catalog berikutnya.
2. Kirim `POST http://localhost:8082/api/v1/catalog/admin/images` dengan **Body → Multipart Form**. Tambah field `file` bertipe **File**, pilih gambar, lalu salin `data.object_key` dari respons `201`.
3. Kirim `POST http://localhost:8082/api/v1/catalog/admin/brands` dengan body JSON berikut. Gunakan nama yang belum ada dan ganti key contoh dengan hasil upload. Respons `201` mengembalikan `data.id` dan `data.status` bernilai `active`.

   ```json
   {
     "name": "Toyota",
     "image_object_key": "catalog/images/abcdefghijklmnopqrstuvwxyz.png"
   }
   ```

4. Kirim `GET http://localhost:8082/api/v1/catalog/admin/brands?search=toy&page=1&limit=20`. Hasil ada di `data` dan informasi pagination ada di `page`.
5. Kirim `PATCH http://localhost:8082/api/v1/catalog/admin/brands/{id}` dengan ID dari langkah create. Respons `200` berisi Brand dengan `data.status` bernilai `inactive`.

   ```json
   {
     "status": "inactive"
   }
   ```

## Coba Banner Hero di Bruno

Setelah migration `000009`, `GET /api/v1/catalog/admin/banners` tetap menampilkan posisi 1, 2, dan 3. Slot kosong mempunyai `content: null`. Pakai base URL Gateway `http://localhost:8080` dan Bearer token OWNER atau ADMIN untuk request di bawah `/admin`.

1. Ambil `data[].id` dari `GET /api/v1/catalog/admin/cars`. Upload gambar Banner untuk mendapatkan `data.object_key`.
2. Kirim `PUT /api/v1/catalog/admin/banners/1` dengan body berikut. Ganti ID dan key contoh dengan nilai dari Catalog.

```json
{
  "car_id": "00000000-0000-4000-8000-000000000001",
  "headline_line_1": "Room for",
  "headline_line_2": "bigger plans.",
  "description": "A spacious SUV for family days and longer drives.",
  "image_object_key": "catalog/images/abcdefghijklmnopqrstuvwxyz.webp"
}
```

3. Kirim `GET /api/v1/catalog/admin/banners/1` dan cek `data.content.car_id`. `GET /api/v1/catalog/home/banners` tanpa token menampilkan slot terisi beserta slug mobilnya. Isi `R2_PUBLIC_BASE_URL` agar API mengirim URL gambar.
4. Setelah mengubah status Car menjadi `inactive`, coba lagi `GET /api/v1/catalog/home/banners`. Buka `GET /api/v1/catalog/cars/{slug}` dan cek `data.status`. Detail tetap merespons 200 dengan status `inactive`.
5. Kirim `DELETE /api/v1/catalog/admin/banners/1`. Respons 200 mengembalikan posisi 1 dengan `content: null`.

Payload lama dengan `brand_id`, `car_model`, dan `car_slug` tetap diterima. Kirim salah satu bentuk payload.

## Coba Car di Bruno

Jalankan Catalog dan Identity. Gunakan Bearer token OWNER atau ADMIN untuk request `/admin`.

1. Salin UUID Brand aktif dari `GET /api/v1/catalog/admin/brands`. Siapkan satu file cover dan minimal satu foto gallery.
2. Untuk setiap file, kirim `POST /api/v1/catalog/admin/images` dengan **Body → Multipart Form** dan field `file` bertipe **File**. Simpan `data.object_key` masing-masing.
3. Kirim `POST http://localhost:8082/api/v1/catalog/admin/cars` dengan JSON berikut. Ganti UUID Brand dan kedua key dengan nilai dari Catalog.

   ```json
   {
     "brand_id": "48a0e5e3-20f5-4e39-a927-01a5b4d5c6e7",
     "model": "Fortuner",
     "trim": "VRZ AT",
     "category": "suv",
     "body_type": "suv",
     "seats": 7,
     "transmission": "automatic",
     "fuel_type": "diesel",
     "description": "A spacious seven-seat SUV for family drives.",
     "daily_rate_idr": 650000,
     "status": "active",
     "cover": {
       "object_key": "catalog/images/abcdefghijklmnopqrstuvwxyz.webp",
       "alt_text": "Toyota Fortuner exterior",
       "label": "Exterior"
     },
     "gallery": [
       {
         "object_key": "catalog/images/zyxwvutsrqponmlkjihgfedcba.webp",
         "alt_text": "Toyota Fortuner interior",
         "label": "Interior"
       }
     ]
   }
   ```

   Respons `201` memuat `data.id`, `data.slug`, status, dan foto. Body di atas langsung memublikasikan Car. Jika field `status` dilewati, status awalnya `inactive`.

4. Untuk edit, kirim `PATCH /api/v1/catalog/admin/cars/{id}`. Body lengkap di atas bisa dipakai lagi. Jika mengubah foto, upload file baru terlebih dahulu. `cover` mengganti cover, dan `gallery` mengganti seluruh gallery. Field yang dilewati dipertahankan. Gunakan `GET /api/v1/catalog/admin/cars/{id}` untuk membaca key gambar yang sudah tersimpan.
5. Coba request berikut dengan **No Auth**.

   - `GET /api/v1/catalog/cars?search=fortuner&seats=7&min_price=400001&max_price=700000&sort=recommended&page=1&limit=20`
   - `GET /api/v1/catalog/cars/{slug}`
   - `GET /api/v1/catalog/cars/{slug}/related?limit=3`
   - `GET /api/v1/catalog/brands`

   Detail publik memuat cover sebagai foto pertama, lalu gallery. Isi `R2_PUBLIC_BASE_URL` agar respons memuat URL gambar publik.

## Coba Featured Cars di Bruno

Setelah migration `000010`, pakai base URL gateway `http://localhost:8080`. Gunakan Bearer token OWNER/ADMIN untuk request di bawah `/admin`.

1. Ambil ID dari `GET /api/v1/catalog/admin/cars`.
2. Kirim `PUT /api/v1/catalog/admin/featured-cars` dengan JSON berikut. Ganti ID contoh dengan ID Car yang ada.

```json
{
  "car_ids": [
    "00000000-0000-4000-8000-000000000001",
    null,
    "00000000-0000-4000-8000-000000000002"
  ]
}
```

3. Cek `GET /api/v1/catalog/admin/featured-cars`. Respons berisi Car dengan `position` 1 dan 3 beserta `status` masing-masing. Posisi 2 kosong.
4. Cek `GET /api/v1/catalog/home/featured-cars` tanpa token. Car active tampil sesuai urutan. Jika salah satunya dinonaktifkan, endpoint publik melewatinya.
5. Untuk mengosongkan daftar, kirim PUT dengan body berikut.

```json
{
  "car_ids": []
}
```

## Coba unit fisik di Bruno

Setelah migration `000011`, pakai base URL gateway `http://localhost:8080` dan Bearer token OWNER atau ADMIN.

1. Ambil ID varian dari `GET /api/v1/catalog/admin/cars`. Ganti `{car_id}` pada URL dengan ID varian tersebut.
2. Kirim `POST /api/v1/catalog/admin/cars/{car_id}/units` dengan body berikut. Respons 201 mengembalikan `data.id` untuk unit dan status default `inactive`.

```json
{
  "internal_code": "INNOVA-001"
}
```

3. Buat unit kedua pada varian yang sama.

```json
{
  "internal_code": "INNOVA-002",
  "status": "active"
}
```

4. Lihat daftar melalui `GET /api/v1/catalog/admin/cars/{car_id}/units?page=1&limit=20`. Tambahkan `&status=active` untuk menyaring unit aktif. Untuk melihat detail, gunakan ID dari respons create di `GET /api/v1/catalog/admin/units/{unit_id}`.
5. Untuk mengubah kode dan menonaktifkan unit kedua, kirim `PATCH /api/v1/catalog/admin/units/{unit_id}` dengan body berikut. Kedua field dapat dikirim terpisah.

```json
{
  "internal_code": "INNOVA-003",
  "status": "inactive"
}
```
