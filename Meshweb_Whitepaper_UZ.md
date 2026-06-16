# Meshweb: Markazlashmagan Saqlash va Hisoblash uchun Peer-to-Peer Protokol

**Versiya 0.1.0 — Iyun 2026**

---

**Annotatsiya.** Zamonaviy bulutli infratuzilma ma'lumotlar va hisoblash resurslarini bir nechta yirik korporatsiyalar qo'lida to'playdi. Bu esa yagona buzilish nuqtalarini, senzura vektorlarini va maxfiylik xavflarini keltirib chiqaradi. Biz Meshweb — to'liq markazlashmagan peer-to-peer fayllarni saqlash protokolini taqdim etamiz. LibP2P va Kademlia DHT asosida qurilgan Meshweb hech qanday markaziy serverga muhtoj emas. Fayllar foydalanuvchi qurilmasida AES-256-GCM bilan shifrlanadi, Reed-Solomon erasure coding (10 ta ma'lumot + 20 ta zaxira shard) yordamida 30 ta bo'lakka bo'linadi va tarmoq bo'ylab taqsimlanadi. 30 ta sharddan istalgan 10 tasi asl faylni qayta tiklash uchun yetarli bo'lib, bu haddan tashqari yuqori xatoga chidamlilikni ta'minlaydi. Tugun identifikatsiyasi o'z-o'zini boshqaruvchi bo'lib, BIP39 mnemonik seed iborasi va Ed25519 kalit juftligidan hosil bo'ladi. Protokol avtonom ishlaydi — hech qanday ro'yxatdan o'tish, hisob yoki markaziy hokimiyat talab qilinmaydi. Kelgusi bosqichlarda MWCoin utility tokeni bilan ishlaydigan hisoblash bozori joriy etiladi.

---

## Mundarija

1. [Kirish va asosiy falsafa](#1-kirish-va-asosiy-falsafa)
2. [Muammo](#2-muammo)
3. [Meshweb yechimi](#3-meshweb-yechimi)
4. [Texnik arxitektura](#4-texnik-arxitektura)
5. [Saqlash protokoli (Joriy etilgan)](#5-saqlash-protokoli-joriy-etilgan)
6. [Identifikatsiya tizimi](#6-identifikatsiya-tizimi)
7. [Ulashish va kontent adreslash](#7-ulashish-va-kontent-adreslash)
8. [Hisoblash bozori (Rejalashtirilgan)](#8-hisoblash-bozori-rejalashtirilgan)
9. [MWCoin tokenomikasi (Rejalashtirilgan)](#9-mwcoin-tokenomikasi-rejalashtirilgan)
10. [Xavfsizlik va tahdid modeli](#10-xavfsizlik-va-tahdid-modeli)
11. [Raqobat tahlili](#11-raqobat-tahlili)
12. [Yo'l xaritasi](#12-yol-xaritasi)
13. [Xulosa](#13-xulosa)

---

## 1. Kirish va asosiy falsafa

Internet dastlab markazlashmagan tarmoq sifatida yaratilgan, ammo zamonaviy infratuzilma markazlashgan bulut provayderlariga qarab og'di. Bu arxitektura uzilishlar, senzura va ma'lumotlar sizib chiqishiga zaifdir.

Meshweb qat'iy asosiy falsafa asosida ishlab chiqilgan:

- **Mutlaq markazlashmaganlik.** Hech qanday markaziy server mavjud emas. NAT o'tkazish uchun ishlatiladigan relay tugunlari almashtiriladigan va har qanday ishtirokchi tomonidan boshqarilishi mumkin.
- **Bardoshlilik.** Protokol yo'q qilib bo'lmaydigan qilib loyihalashtirilgan. Ma'lumotlar hosting tugunlarning 66% bir vaqtning o'zida o'chib qolsa ham saqlanib qoladi.
- **Avtonomiya.** Meshweb inson aralashuvisiz to'liq ishlaydi, hatto yaratuvchilari yo'qligida ham o'z faoliyatini davom ettiradi.
- **Progressiv kuch.** Tarmoqning sig'imi, ortiqchaligi va yo'naltirish samaradorligi har bir yangi tugun qo'shilishi bilan oshadi.
- **Odatiy holda maxfiylik.** Barcha ma'lumotlar foydalanuvchi qurilmasini tark etishidan oldin shifrlanadi. Hech bir tugun — shu jumladan ma'lumotlarni saqlovchi tugun ham — uni o'qiy olmaydi.

---

## 2. Muammo

Bugungi ma'lumotlarni saqlash va hisoblash bozorlari jiddiy to'siqlarga duch kelmoqda:

1. **Monopollashtirish.** Bir nechta provayder (AWS, Google Cloud, Azure) narxlarni, shartlarni va kirishni belgilaydi.
2. **Senzura.** Markazlashgan provayderlar bir tomonlama kontentni o'chirib tashlashi, hisoblarni muzlatishi yoki olib tashlash so'rovlarini bajarishi mumkin.
3. **Maxfiylik.** Foydalanuvchilar shifrlanmagan ma'lumotlar va meta-ma'lumotlarni uchinchi tomonlarga ishonib topshirishga majbur.
4. **Yagona buzilish nuqtasi.** Bitta provayderdagi mintaqaviy uzilish millionlab xizmatlarni o'chirib qo'yishi mumkin.
5. **Resurslar samarasizligi.** Milliardlab iste'molchi qurilmalari foydalanilmagan bo'sh saqlash va hisoblash quvvatlariga ega.

---

## 3. Meshweb yechimi

Meshweb har bir qurilmani global, ruxsatsiz saqlash tarmog'idagi tugunga aylantiradi. Protokol shifrlash, fragmentatsiya, taqsimlash, kashfiyot va qayta tiklanishni to'liq avtonom boshqaradi.

```
┌──────────────────────────────────────────────────────────┐
│                    MESHWEB PROTOKOLI                      │
│                                                          │
│  ┌─────────┐   ┌─────────┐   ┌─────────┐   ┌─────────┐ │
│  │Foydalanu-│   │ AES-256 │   │  Reed-  │   │   P2P   │ │
│  │chi fayli │──▶│   GCM   │──▶│ Solomon │──▶│  To'ri  │ │
│  │         │   │ Shifrlash│   │ 10 + 20 │   │  (DHT)  │ │
│  └─────────┘   └─────────┘   └─────────┘   └─────────┘ │
│                                                          │
│  Yuklash: Fayl → Shifrlash → Bo'lish → Tarqatish        │
│  Olish: Topish → 10/30 olish → Tiklash → Deshifrlash    │
└──────────────────────────────────────────────────────────┘
```

---

## 4. Texnik arxitektura

Meshweb arxitekturasi har biri ma'lum bir vazifani bajaruvchi alohida qatlamlarga bo'lingan.

### 4.1 1-qatlam: Transport (LibP2P)

Meshwebning asosi — LibP2P asosida qurilgan mustahkam, senzuraga chidamli peer-to-peer tarmoq.

| Komponent | Texnologiya | Maqsad |
|---|---|---|
| TCP Transport | LibP2P TCP | Asosiy ulanish |
| WebSocket | LibP2P WS | Brauzerga mos transport |
| QUIC | LibP2P QUIC-v1 | Kechikishi past UDP transport |
| NAT o'tkazish | AutoRelay + HolePunching | NAT orqasidagi ulanish |
| Relay | Circuit Relay v2 | Simmetrik NATlar uchun zaxira |
| Shifrlash | TLS + Noise Protocol | Barcha tugunlararo trafik shifrlangan |

**NAT o'tkazish strategiyasi:**
1. Tugun TCP/QUIC orqali to'g'ridan-to'g'ri ulanishga harakat qiladi.
2. Agar bloklansa, AutoRelay mavjud relay tugunlari bilan faollashadi.
3. To'g'ridan-to'g'ri peer-to-peer yo'l uchun HolePunching uriniladi.
4. Agar barchasi muvaffaqiyatsiz bo'lsa, Circuit Relay v2 kafolatlangan ulanishni ta'minlaydi.

Tarmoqdagi har qanday tugun relay sifatida xizmat qilishi mumkin. Imtiyozli infratuzilma mavjud emas.

### 4.2 2-qatlam: Kashfiyot (Kademlia DHT)

Tugunlarni topish va kontentni yo'naltirish uchun **Kademlia tarqatilgan xesh jadvali** ishlatiladi.

- **Tugunlarni topish:** Tugunlar DHT yo'naltirish kashfiyoti orqali `meshweb-network` nom makonida o'zlarini e'lon qiladi.
- **Resurs e'lonlari:** Tugunlar har 5 soniyada GossipSub orqali `meshweb-nodes` mavzusida CPU va RAM mavjudligini uzatadi.
- **Kontent yo'naltirish:** Fayl shardlari DHT orqali e'lon qilinadi va topiladi, bu har qanday tugunga markaziy indekslashsiz ma'lumotlarni topish va olish imkonini beradi.
- **Bootstrap tozalovchi:** Har 30 soniyada ishlaydigan fon jarayoni bootstrap tugunlarini tiriklik uchun sinab ko'radi va erishib bo'lmaydigan tugunlarni mahalliy bootstrap ro'yxatidan olib tashlaydi.

### 4.3 3-qatlam: Xabar almashish (GossipSub)

Meshweb real vaqtdagi tarmoq aloqasi uchun **GossipSub** (LibP2P pubsub protokoli) dan foydalanadi.

| Mavzu | Maqsad |
|---|---|
| `meshweb-nodes` | Resurs e'lonlari (CPU, RAM) |
| `meshweb-jobs` | Hisoblash vazifalarini uzatish (rejalashtirilgan) |
| `meshweb-results` | Vazifa natijalarini xabar qilish (rejalashtirilgan) |

### 4.4 4-qatlam: Saqlash (Joriy etilgan)

[5-bo'limda](#5-saqlash-protokoli-joriy-etilgan) batafsil.

### 4.5 5-qatlam: Hisoblash bozori (Rejalashtirilgan)

[8-bo'limda](#8-hisoblash-bozori-rejalashtirilgan) batafsil.

---

## 5. Saqlash protokoli (Joriy etilgan)

Saqlash qatlami to'liq joriy etilgan va ishlamoqda. U shifrlangan, ortiqcha, markazlashmagan fayl saqlashni ta'minlaydi.

### 5.1 Yuklash quvuri (Upload Pipeline)

```
Asl fayl (N bayt)
        │
        ▼
┌───────────────────┐
│ AES-256-GCM       │  Tasodifiy 256-bitli kalit yaratiladi
│ Mijoz tomonida    │  12-baytli nonce shifrlangan matnga qo'shiladi
│ Shifrlash         │  Natija: nonce || shifrlangan matn || auth tag
└───────┬───────────┘
        │
        ▼ shifrlangan matn (N + 28 bayt)
┌───────────────────┐
│ SHA-256 xesh       │  CID orqali kontent-adreslangan (IPFS-mos)
│ CID yaratish       │  Multihash: SHA2-256, Codec: Raw
└───────┬───────────┘
        │
        ▼
┌───────────────────┐
│ Reed-Solomon       │  10 ma'lumot shard + 20 zaxira shard = jami 30
│ Erasure Coding     │  Istalgan 10/30 shard → to'liq tiklash
│ (10, 20)           │  Chidamlilik: 66% shard yo'qolishi
└───────┬───────────┘
        │
        ▼ 30 shard
┌───────────────────┐
│ Mahalliy saqlash   │  Har bir shard: storage/{CID}/shard_{0..29}
│ + DHT e'lon        │  CID kashfiyot uchun DHT ga e'lon qilinadi
└───────────────────┘
        │
        ▼
┌───────────────────┐
│ .meshweb metadata  │  JSON fayl: versiya, fayl nomi, CID,
│ Fayl yaratish      │  AES kalit, shard soni, asl hajm, yaratuvchi ID
└───────────────────┘
```

### 5.2 Yuklab olish quvuri (Download Pipeline)

```
meshweb:// havola yoki .meshweb fayl
        │
        ▼
┌───────────────────┐
│ Metadata tahlili   │  Ajratish: CID, AES kalit, fayl nomi, asl hajm
└───────┬───────────┘
        │
        ▼
┌───────────────────┐
│ DHT + PubSub       │  Fayl shardlarini saqlovchi tugunlarni topish
│ Provayder topish   │  Protokol: /meshweb/storage/1.0.0
└───────┬───────────┘
        │
        ▼
┌───────────────────┐
│ Shard olish        │  LibP2P oqimlari orqali shardlarni so'rash
│ (30 dan 10 tasi)  │  JSON-line protokoli: So'rov → Javob
│                    │  Ma'lumotlar Base64 kodlangan baytlar sifatida
└───────┬───────────┘
        │
        ▼
┌───────────────────┐
│ Reed-Solomon       │  Istalgan 10 sharddan to'liq shifrlangan matnni
│ Qayta tiklash      │  tiklash. OriginalSize ga qirqish (RS paddingni
│                    │  olib tashlash)
└───────┬───────────┘
        │
        ▼
┌───────────────────┐
│ AES-256-GCM       │  Metadatadan olingan kalit bilan deshifrlash
│ Deshifrlash        │  Autentifikatsiya tegini tekshirish (butunlik)
└───────┬───────────┘
        │
        ▼
   Asl fayl
```

### 5.3 Oqim protokoli spetsifikatsiyasi

**Protokol ID:** `/meshweb/storage/1.0.0`

**So'rov (JSON-line):**
```json
{
  "file_id": "bafkreie...",
  "shard": 0
}
```

**Javob (JSON-line):**
```json
{
  "file_id": "bafkreie...",
  "shard": 0,
  "data": "<base64-kodlangan shard baytlari>",
  "error": ""
}
```

### 5.4 Reed-Solomon parametrlari

| Parametr | Qiymat | Asoslanishi |
|---|---|---|
| Ma'lumot shardlari | 10 | Tiklash uchun zarur bo'lgan minimal fragmentlar |
| Zaxira shardlari | 20 | Ortiqchalik fragmentlari |
| Jami shardlar | 30 | Tarqatilgan jami fragmentlar |
| Xatoga chidamlilik | 66.7% | 30 dan 20 tagacha shard yo'qolishi mumkin |
| Saqlash ortiqchaligi | 3x | Har bir fayl tarmoq bo'ylab o'z asl hajmidan 3 baravar joy egallaydi |

### 5.5 Shifrlash spetsifikatsiyasi

| Parametr | Qiymat |
|---|---|
| Algoritm | AES-256-GCM (Galois/Counter Mode) |
| Kalit hajmi | 256 bit (32 bayt), kriptografik tasodifiy |
| Nonce hajmi | 96 bit (12 bayt), kriptografik tasodifiy |
| Autentifikatsiya | O'rnatilgan GCM auth tag (128 bit) |
| Kalit hosil qilish | Yo'q — har bir fayl uchun toza tasodifiy kalit |
| Kalit saqlash | `.meshweb` metadata va `meshweb://` havolalarga joylashtirilgan |

### 5.6 Kontent adreslash

Fayllar **CID v1** (IPFS-mos) yordamida kontent-adreslangan:

- **Xesh funksiyasi:** SHA2-256
- **Codec:** Raw (0x55)
- **Multihash formati:** Standart multihash kodlash
- **Misol:** `bafkreie7ohyl7zg6g5wxhvzah5kkgbq...`

Bu Meshweb saqlashni kengroq IPFS kontent-adreslash ekotizimi bilan mos qiladi.

---

## 6. Identifikatsiya tizimi

Meshweb **o'z-o'zini boshqaruvchi identifikatsiya** tizimini ro'yxatdan o'tish yoki markaziy hokimiyatsiz amalga oshiradi.

### 6.1 Kalit yaratish

```
BIP39 Entropiya (128 bit)
        │
        ▼
┌───────────────────┐
│ 12 so'zli          │  Standart BIP39 so'z ro'yxati
│ Mnemonik ibora     │  Inson o'qiy oladigan zaxira
└───────┬───────────┘
        │
        ▼
┌───────────────────┐
│ BIP39 Seed         │  512-bitli deterministik seed
│ Hosil qilish       │  Bo'sh parol iborasi bilan PBKDF2
└───────┬───────────┘
        │
        ▼ birinchi 32 bayt
┌───────────────────┐
│ Ed25519 kalit      │  Seeddan deterministik
│ juftligi           │  Maxfiy kalit + Ochiq kalit
└───────┬───────────┘
        │
        ▼
┌───────────────────┐
│ LibP2P Peer ID     │  Ochiq kalitdan hosil bo'ladi
│                    │  Tarmoqda yagona identifikator
└───────────────────┘
```

### 6.2 Xususiyatlar

| Xususiyat | Amalga oshirish |
|---|---|
| Mnemonik | BIP39, 12 so'z, 128-bitli entropiya |
| Kalit algoritmi | Ed25519 |
| Peer ID | Ochiq kalitdan hosil bo'ladi |
| Saqlash | Mahalliy ilova ma'lumotlari katalogida shifrlangan JSON |
| Zaxira | Seed ibora yoki identifikatsiya faylini eksport qilish |
| Tiklash | 12 so'zdan to'liq identifikatsiyani tiklash |
| Ko'chirish | Xuddi shu seed ibora → har qanday qurilmada xuddi shu identifikatsiya |

### 6.3 Xavfsizlik

- Maxfiy kalitlar cheklangan fayl ruxsatlari bilan mahalliy sifatida saqlanadi.
- Seed ibora asosiy sir hisoblanadi — uni yo'qotish identifikatsiyani butunlay yo'qotishni anglatadi.
- Markaziy reestr mavjud emas. Identifikatsiya egaligi kriptografik tarzda isbotlanadi.

---

## 7. Ulashish va kontent adreslash

Meshweb fayllarni ulashish uchun ikki mexanizmni taqdim etadi:

### 7.1 Meshweb havolalari

```
meshweb://file/{CID}?k={AES_KALIT_HEX}&n={FAYL_NOMI_BASE64}&s={ASL_HAJM}
```

| Parametr | Tavsif |
|---|---|
| `CID` | Kontent identifikatori (SHA-256 asosida) |
| `k` | AES-256 deshifrlash kaliti (hex-kodlangan) |
| `n` | Asl fayl nomi (Base64 URL-kodlangan) |
| `s` | Asl shifrlangan matn hajmi (RS paddingni olib tashlash uchun) |

Bu havolalar o'z-o'zidan yetarli: havolaga ega bo'lgan har qanday kishi faylni hech qanday hisob yoki ro'yxatdan o'tishsiz yuklab olishi va deshifrlashi mumkin.

### 7.2 .meshweb fayllari

`.meshweb` fayli quyidagilarni o'z ichiga olgan JSON metadata fayli:

```json
{
  "version": "1.0",
  "file_name": "hujjat.pdf",
  "file_size": 1048576,
  "original_size": 1048604,
  "file_id": "bafkreie...",
  "shards": 30,
  "min_shards": 10,
  "encryption": "AES-256-GCM",
  "key_hash": "a1b2c3...",
  "aes_key": "<hex-kodlangan kalit>",
  "created_at": "<ISO 8601 vaqt tamg'asi>",
  "creator_id": "<peer ID>"
}
```

Windows da `.meshweb` fayllari fayl assotsiatsiyasi sifatida ro'yxatdan o'tkazilishi mumkin, bu Meshweb GUI da ikki marta bosish orqali to'g'ridan-to'g'ri ochish imkonini beradi.

---

## 8. Hisoblash bozori (Rejalashtirilgan)

Hisoblash bozori loyihalashtirilgan, ammo hali joriy etilmagan. U tugunlarga GPU, CPU va RAM resurslarini boshqa ishtirokchilarga ijaraga berish imkonini beradi.

### 8.1 Rejalashtirilgan arxitektura

- **Ijara protokoli:** `/meshweb/rent/1.0.0` (oqim asosida muzokaralar)
- **Resurslarni topish:** `meshweb-nodes` da GossipSub e'lonlari
- **Vazifa hayot sikli:** So'rov → Qabul/Rad → Bajarish → Hisob-kitob
- **Sandboxing:** Ish yukini bajarish uchun Docker va/yoki WebAssembly izolyatsiyasi
- **Narxlash:** Algoritmik talab-taklif asosida narxlash

### 8.2 Hozirgi holat

Protokol skeleti kodda mavjud:
- `RentalJob` va `RentRequest` ma'lumotlar tuzilmalari aniqlangan
- `/meshweb/rent/1.0.0` uchun oqim boshqaruvchisi joriy etilgan
- Peer-to-peer muzokaralar oqimi (so'rov → javob) ishlaydi
- To'lov sikli infratuzilmasi mavjud (hozirda o'chirilgan)

To'liq joriy etish to'lov hisob-kitoblari uchun MWCoin integratsiyasini kutmoqda.

---

## 9. MWCoin tokenomikasi (Rejalashtirilgan)

Meshweb hisoblash va saqlash bozorini quvvatlash uchun mo'ljallangan **MWCoin** utility tokenida ishlaydi.

### 9.1 Token taqsimoti (100M qattiq cheklangan)

| Ajratma | Foiz | Miqdor | Maqsad |
|---|---|---|---|
| Mining va tugun mukofotlari | 75% | 75,000,000 | Hisoblash/saqlash ta'minlovchi tugunlarga beriladi |
| Protokol fondi | 10% | 10,000,000 | Infratuzilmani rivojlantirish va texnik xizmat |
| Boshlang'ich likvidlik | 10% | 10,000,000 | DEX bozor yaratish |
| Genesis hissa qo'shuvchilar | 5% | 5,000,000 | 24 oy muddatga smart-kontraktda qulflangan |

### 9.2 Taxminiy daromadlar

| Resurs | Davomiylik | Taxminiy daromad |
|---|---|---|
| Yuqori darajali GPU (RTX 4090 / A100) | 1 soat | ~0.5 MWCoin |
| Standart CPU (8 yadroli) | 1 soat | ~0.05 MWCoin |
| 1 TB shifrlangan saqlash | 1 oy | ~2.0 MWCoin |

*Daromadlar tarmoq taklif va talabi asosida algoritmik tarzda o'zgartiriladi.*

### 9.3 Hisob-kitob mexanizmlari

- **Zarb qilish:** MWCoin faqat samarali foydalanish orqali zarb qilinadi (saqlash ta'minlash, hisoblashni bajarish).
- **Depozit:** Smart-kontraktlar vazifani bajarish davomida xaridor mablag'larini ushlab turadi.
- **Protokol solig'i:** Zarb qilingan tangalarning 2-5% Protokol fondiga yo'naltiriladi (qattiq kodlangan, o'zgarmas).
- **Almashish:** MWCoin DEXlarda fiat yoki boshqa kriptovalyutalarga almashtirilishi mumkin.

---

## 10. Xavfsizlik va tahdid modeli

### 10.1 Ma'lumotlar xavfsizligi

| Tahdid | Qarshi chora |
|---|---|
| Saqlash tugunida ma'lumotlar sizib chiqishi | AES-256-GCM shifrlash — tugunlar faqat shifrlangan matn fragmentlarini saqlaydi |
| Kalit tutib olish | Kalitlar foydalanuvchi tomonidan tashqi kanallar orqali ulashilgan havolalar/fayllarga joylashtirilgan |
| Ma'lumotlar buzilishi | GCM autentifikatsiya tegi har qanday o'zgartirishni aniqlaydi |
| Ommaviy tugun buzilishi | Reed-Solomon 66.7% shard yo'qolishiga chidaydi |

### 10.2 Tarmoq xavfsizligi

| Tahdid | Qarshi chora |
|---|---|
| Sybil hujumi | Resursga asoslangan obro' (kelajakda: MWCoin staking) |
| Eclipse hujumi | Bir nechta bootstrap tugunlari, DHT asosida turli xil yo'naltirish |
| O'rtada turgan odam (MITM) | Barcha LibP2P ulanishlari TLS/Noise shifrlashdan foydalanadi |
| DPI / Senzura | QUIC transport, WebSocket qo'llab-quvvatlash, relay zaxirasi |
| Relay buzilishi | Relay faqat shifrlangan trafikni ko'radi; ma'lumotlar yoki shardlarni deshifray olmaydi |

### 10.3 Identifikatsiya xavfsizligi

| Tahdid | Qarshi chora |
|---|---|
| Identifikatsiya o'g'irlash | Ed25519 maxfiy kalit cheklangan ruxsatlar bilan mahalliy saqlanadi |
| Kalit yo'qolishi | BIP39 seed ibora har qanday qurilmada to'liq tiklash imkonini beradi |
| Soxta ko'rinish | Peer ID kriptografik tarzda Ed25519 ochiq kalitga bog'langan |

### 10.4 Ma'lum cheklovlar (v0.1.0)

- **Shard replikatsiya protokoli yo'q.** Hozirda shardlar faqat yuklovchining tugunida saqlanadi. Ko'p tugunli taqsimlash uchun yuklovchi onlayn bo'lishi kerak.
- **Saqlash uchun rag'bat yo'q.** Tugunlar o'z fayllarini saqlaydi, ammo boshqalarning ma'lumotlarini saqlash uchun iqtisodiy rag'bat yo'q (MWCoin kutilmoqda).
- **Ma'lumotlar doimiyligi kafolati yo'q.** Agar yuklovchi tugun butunlay o'chib qolsa va boshqa hech qanday tugun shardlarga ega bo'lmasa, fayl yo'qoladi.

---

## 11. Raqobat tahlili

| Xususiyat | Meshweb | Filecoin | IPFS | Akash | io.net |
|---|---|---|---|---|---|
| Markazlashmagan saqlash | ✅ | ✅ | ✅ | ❌ | ❌ |
| Markazlashmagan hisoblash | 🔜 Reja | ❌ | ❌ | ✅ | ✅ |
| Mijoz tomonida shifrlash | ✅ Odatiy | ❌ Ixtiyoriy | ❌ | ❌ | ❌ |
| Erasure Coding | ✅ RS(10,20) | ✅ | ❌ | ❌ | ❌ |
| Ro'yxatdan o'tishsiz | ✅ | ❌ | ✅ | ❌ | ❌ |
| O'z-o'zini boshqaruvchi ID | ✅ BIP39 | ❌ | ❌ | ❌ | ❌ |
| Desktop GUI | ✅ | ❌ | ✅ | ❌ | ❌ |
| Yengil dastur | ✅ ~30MB | ❌ Og'ir | ❌ Og'ir | ❌ | ❌ |
| Markaziy koordinator | ❌ Yo'q | Qisman | ❌ Yo'q | Qisman | ✅ Talab qilinadi |

**Asosiy farqlari:**
- **Filecoin bilan taqqoslaganda:** Filecoin hisoblash jihatdan og'ir Proof-of-Spacetime talab qiladi. Meshweb har qanday iste'molchi qurilmasi uchun yetarlicha yengil.
- **IPFS bilan taqqoslaganda:** IPFS kontent adreslashni ta'minlaydi, ammo shifrlash, erasure coding va hisoblash bozori yo'q.
- **Akash/io.net bilan taqqoslaganda:** Bular hisoblashga e'tibor qaratadi, ammo markaziy koordinatorlarga tayanadi. Meshweb 100% serversiz.

---

## 12. Yo'l xaritasi

### 1-bosqich: Genesis ✅ (Hozirgi — v0.1.0)
- [x] Asosiy P2P protokoli (LibP2P + Kademlia DHT)
- [x] Shifrlangan fayl saqlash (AES-256-GCM + Reed-Solomon)
- [x] O'z-o'zini boshqaruvchi identifikatsiya (BIP39 + Ed25519)
- [x] Windows uchun desktop GUI (Wails + React)
- [x] Ko'p tilli interfeys (ingliz, o'zbek, rus)
- [x] `meshweb://` havola ulashish va `.meshweb` fayl assotsiatsiyasi
- [x] GitHub da ochiq kodli nashr

### 2-bosqich: Bardoshlilik (v0.2.0)
- [ ] Ko'p tugunli shard taqsimlash (tarmoq tugunlari bo'ylab shardlarni saqlash)
- [ ] Shard replikatsiya protokoli (tugun chiqib ketganda avtomatik qayta replikatsiya)
- [ ] Mac va Linux desktop dasturlari
- [ ] Fayllarni mahkamlash va doimiylik kafolatlari
- [ ] Kengaytirilgan relay infratuzilmasi

### 3-bosqich: Iqtisodiyot (v0.3.0)
- [ ] MWCoin mainnet ishga tushirish
- [ ] Hisoblash bozorini faollashtirish
- [ ] Saqlash rag'bat tizimi (shardlarni saqlash uchun MWCoin ishlash)
- [ ] Smart-kontraktga asoslangan hisob-kitob
- [ ] DEX likvidlik ta'minlash

### 4-bosqich: Masshtab (v1.0.0)
- [ ] Mobil ilovalar (Android, iOS)
- [ ] GPU hisoblash orkestratsiyasi (AI/ML ish yuklari)
- [ ] Markazlashmagan boshqaruv
- [ ] Avtomatlashtirilgan fiat shlyuzlari
- [ ] Dasturchilar uchun SDK va API

---

## 13. Xulosa

Meshweb v0.1.0 ishlaydigan, ishlab chiqarishga tayyor markazlashmagan fayl saqlash protokolini taqdim etadi. Fayllar shifrlangan, fragmentlangan va kontent-adreslangan — hech qanday markaziy server foydalanuvchi ma'lumotlariga tegmaydi. Identifikatsiya o'z-o'zini boshqaruvchi bo'lib, oddiy 12 so'zlik seed iborasidan hosil bo'ladi.

Bu Genesis nashri. Asos qo'yildi. Keyingi bosqichlar — ko'p tugunli taqsimlash, iqtisodiy rag'batlar va hisoblash bozorlari — Meshwebni saqlash protokolidan global, ruxsatsiz infratuzilma qatlamiga aylantiradi.

Tarmoq hech kimga tegishli emas. U hamma uchun ishlaydi. Va har bir yangi tugun bilan yanada kuchliroq bo'ladi.

---

*Meshweb MIT litsenziyasi ostida chiqarilgan ochiq kodli dasturiy ta'minot.*
*Repozitoriy: [github.com/meshweb-protocol/meshweb](https://github.com/meshweb-protocol/meshweb)*
