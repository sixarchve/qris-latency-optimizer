-- TABEL MERCHANTS
-- Menyimpan data merchant/toko pemilik QRIS
CREATE TABLE merchants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    qr_id VARCHAR(50) UNIQUE NOT NULL, 
    merchant_name VARCHAR(100) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexing pada qr_id agar pencarian data saat QR di-scan selalu cepat
CREATE INDEX idx_merchants_qr_id ON merchants(qr_id);

-- TABEL TRANSACTIONS
-- Menyimpan riwayat transaksi pembayaran
CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id UUID REFERENCES merchants(id),
    amount DECIMAL(15, 2) NOT NULL,
    status VARCHAR(20) DEFAULT 'PENDING',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- DATA AWAL SIMULASI (DUMMY DATA)
-- Data ini akan otomatis terisi saat Docker Compose dijalankan
INSERT INTO merchants (qr_id, merchant_name) VALUES 
('000201010211265700', 'Kantin FILKOM UB'),
('000201010211265701', 'Toko Buku Mitra Niaga'),
('000201010211265702', 'Kopi Rural Sejahtera');