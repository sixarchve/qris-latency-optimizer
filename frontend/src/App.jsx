import { useEffect, useState } from "react";
import { QRCodeCanvas } from "qrcode.react";

function parseQrisPayload(payload) {
  const result = { merchantName: "", city: "" };
  if (!payload) return result;

  let i = 0;
  while (i < payload.length - 4) {
    const tag = payload.substring(i, i + 2);
    const len = parseInt(payload.substring(i + 2, i + 4), 10);
    if (Number.isNaN(len)) break;

    const value = payload.substring(i + 4, i + 4 + len);

    if (tag === "59") result.merchantName = value;
    if (tag === "60") result.city = value;

    i += 4 + len;
  }

  return result;
}

export default function App() {
  const [payload, setPayload] = useState("");
  const [merchant, setMerchant] = useState({ merchantName: "", city: "" });
  const [loading, setLoading] = useState(true);
  const [inputAmount, setInputAmount] = useState(1000);
  const [submittedAmount, setSubmittedAmount] = useState(1000);

  useEffect(() => {
    if (!submittedAmount || submittedAmount <= 0) {
      setPayload("");
      return;
    }
    setLoading(true);
    fetch("/api/qris?amount=" + submittedAmount)
      .then(res => res.json())
      .then(data => {
        setPayload(data.qris_payload);
        setMerchant(parseQrisPayload(data.qris_payload));
        setLoading(false);
      })
      .catch(err => {
        console.error("Failed to fetch QRIS payload", err);
        setLoading(false);
      });
  }, [submittedAmount]);

  const formatRupiah = new Intl.NumberFormat("id-ID", {
    style: "currency",
    currency: "IDR",
    minimumFractionDigits: 0,
  });

  return (
    <div style={styles.page}>
      <div style={styles.card}>
        <div style={styles.header}>
          <div>
            <p style={styles.qrisText}>QRIS PAYMENT</p>
            <h1 style={styles.title}>Capstone Pay</h1>
          </div>
          <div style={styles.liveBadge}>
            <span style={styles.dot}></span> Live
          </div>
        </div>

        <div style={styles.merchant}>
          <h2 style={{ color: "black" }}>{merchant.merchantName}</h2>
          <p style={{ color: "black" }}>{merchant.city}</p>
        </div>

        <div style={styles.amountBox}>
          <p style={{ color: "black", fontSize: "12px", fontWeight: "bold" }}>ENTER AMOUNT (IDR)</p>
          <div style={{ display: 'flex', gap: '8px', marginTop: '8px' }}>
            <input
              type="number"
              value={inputAmount}
              onChange={(e) => setInputAmount(e.target.value === '' ? '' : Number(e.target.value))}
              style={styles.amountInput}
              min="1"
            />
            <button
              onClick={() => setSubmittedAmount(inputAmount)}
              style={styles.generateButton}
            >
              Generate
            </button>
          </div>
        </div>

        <div style={styles.qrWrapper}>
          {loading ? (
            <p>Generating QR...</p>
          ) : (
            <>
              <QRCodeCanvas value={payload} size={220} />
              <div style={styles.scanLine}></div>
            </>
          )}
        </div>

        <p style={styles.instruction}>
          Scan this QR to pay using any QRIS-supported app
        </p>

        <div style={styles.logoRow}>
          {["QRIS Supported", "All Banks", "E-Wallet"].map((b) => (
            <span key={b} style={styles.logo}>
              {b}
            </span>
          ))}
        </div>
      </div>
    </div>
  );
}

const styles = {
  page: {
    minHeight: "100vh",
    display: "flex",
    justifyContent: "center",
    alignItems: "center",
    background: "#f4f4f4",
  },
  card: {
    width: "100%",
    maxWidth: "360px",
    borderRadius: "24px",
    background: "white",
    overflow: "hidden",
    boxShadow: "0 15px 40px rgba(0,0,0,0.12)",
  },
  header: {
    background: "linear-gradient(135deg, #ef4444, #b91c1c)",
    color: "white",
    padding: "20px",
    display: "flex",
    justifyContent: "space-between",
  },
  qrisText: {
    fontSize: "11px",
    letterSpacing: "2px",
  },
  title: {
    marginTop: "6px",
    fontSize: "36px",
    lineHeight: "1.2",
    maxWidth: "200px",
  },
  liveBadge: {
    background: "rgba(255,255,255,0.2)",
    padding: "6px 12px",
    borderRadius: "20px",
    fontSize: "12px",
    alignSelf: "flex-start"
  },
  dot: {
    height: "6px",
    width: "6px",
    background: "white",
    borderRadius: "50%",
    display: "inline-block",
    marginRight: "6px",
  },
  merchant: {
    textAlign: "center",
    padding: "20px",
  },
  amountBox: {
    margin: "0 20px",
    padding: "16px",
    background: "#fce7e7",
    borderRadius: "16px",
    textAlign: "center",
  },
  amountInput: {
    flex: 1,
    fontSize: "20px",
    fontWeight: "bold",
    textAlign: "center",
    width: "100%",
    border: "none",
    background: "white",
    borderRadius: "8px",
    color: "black",
    outline: "none",
    padding: "8px"
  },
  generateButton: {
    padding: "8px 16px",
    background: "#ef4444",
    color: "white",
    border: "none",
    borderRadius: "8px",
    fontWeight: "bold",
    cursor: "pointer",
    boxShadow: "0 4px 6px rgba(239, 68, 68, 0.3)",
  },
  qrWrapper: {
    margin: "20px",
    padding: "16px",
    background: "#fff",
    borderRadius: "16px",
    border: "1px solid #eee",
    position: "relative",
    textAlign: "center",
  },
  scanLine: {
    position: "absolute",
    width: "100%",
    height: "3px",
    background: "rgba(255,0,0,0.4)",
    top: "50%",
    animation: "scan 2s infinite",
  },
  instruction: {
    textAlign: "center",
    fontSize: "12px",
    color: "#666",
  },
  logoRow: {
    display: "flex",
    justifyContent: "center",
    gap: "10px",
    margin: "20px",
    flexWrap: "wrap",
  },
  logo: {
    fontSize: "10px",
    background: "#eee",
    padding: "6px 10px",
    borderRadius: "12px",
  },
};