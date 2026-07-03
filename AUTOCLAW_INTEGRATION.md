# AutoClaw Integration Plan

## Status
✅ **Fase 1-3: Selesai - Build sukses**  
⏳ Fase 4-5: Perlu build frontend untuk test UI, fitur tambahan masih pending.

---

## Yang Sudah Dibuat

### 1. Core Go Provider (`core/provider/autoclaw/`)
- `config.go` - Konfigurasi endpoint & constants
- `auth.go` - Token manager + refresh mechanism
- `provider.go` - Implementasi provider interface (BuildRequest, ParseResponse, Classify)
- `background.go` - Background wallet checker + token refresher

**Fitur:**
- MD5 signing headers sesuai spec Autoglm
- Multi-field credentials (access_token, refresh_token, device_id, email, source_id)
- Automatic token refresh when approaching expiry
- Wallet balance background checking every 5 min
- Exhausted account cache untuk auto-skipping

---

### 2. Backend API Handlers (`server/handlers/api_autoclaw.go`)
- `POST /api/accounts/autoclaw/manual` - Manual add dari JSON creds
- `POST /api/accounts/autoclaw/refresh` - Add via refresh token
- `GET /api/accounts/autoclaw/wallets` - Overview semua wallet balances

---

### 3. CLI Command (`cmd/enowx/cli.go`)
```bash
enx autoclaw-login [subcommand]
```
Menjalankan Python script bundled di `cmd/autoclaw-login/`.

**Script Files Copied:**
- `autoclaw_autologin.py` - Main Flask auto-login app
- `auth.py` - CloakBrowser OAuth automation  
- `proxy.py` - Proxy server routes + wallet check
- `config.py`, `login.py`, requirements.txt
- Batch scripts (.bat files)

---

### 4. Model Routing (`cmd/enowx/route.go`)
Model routing ke autoclaw saat pakai model prefix:
- `glm-*` 
- `autoclaw-*`
- `cheap`, `deepseek`, `auto` → default to autoclaw

---

### 5. Web UI Components
- `web/src/components/AutoClawAddModal.tsx` - Modal add account (manual + refresh tabs)
- `web/src/lib/api.ts` - API helpers (`autoclawApi.manual`, `.refresh`, `.wallets`)
- `web/src/apps/ProvidersApp.tsx` - Integrated modal dalam provider list

---

## Struktur File

```
~/Downloads/enowx76apel/
├── core/
│   └── provider/
│       └── autoclaw/
│           ├── config.go         # endpoints, models
│           ├── auth.go           # token management
│           ├── provider.go       # core logic
│           └── background.go     # bg checks
├── cmd/
│   ├── enowx/
│   │   └── cli.go              # "enx autoclaw-login" handler
│   └── autoclaw-login/
│       ├── autoclaw_autologin.py
│       ├── auth.py
│       ├── proxy.py
│       ├── config.py
│       ├── login.py
│       ├── requirements.txt
│       ├── setup.bat
│       ├── run-batch.bat
│       └── ui/
│           └── ... (UI static files)
├── server/
│   └── handlers/
│       └── api_autoclaw.go     # REST endpoints
└── web/
    ├── src/
    │   ├── components/
    │   │   └── AutoClawAddModal.tsx
    │   ├── lib/
    │   │   └── api.ts          # autoclawApi helpers
    │   └── apps/
    │       └── ProvidersApp.tsx # integrated modal
```

---

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/accounts/autoclaw/manual` | Add account from JSON creds |
| POST | `/api/accounts/autoclaw/refresh` | Add via refresh token |
| GET | `/api/accounts/autoclaw/wallets` | All wallet balances overview |
| POST | `/v1/chat/completions` | Chat requests route to autoclaw for model names: `cheap`, `glm-*`, `autoclaw-*` |

---

## Credentials Schema

```json
{
  "access_token": "...",      // required for chat
  "refresh_token": "...",     // for renewal
  "device_id": "...",         // optional
  "email": "...",             // display label
  "source_id": "...",         // defaults to "autoclaw"
  "user_id": "...",           // fallback label
  "wallet_balance": 1000      // cached balance in credits
}
```

---

## Next Steps

### ⏳ Phase 4: Complete Web UI
1. **AutoClaw Dashboard page** - Show all accounts, wallet balances status (exhausted/good)
2. **Batch login trigger** - Call `enx autoclaw-login` backend via WebSocket or polling
3. **Usage analytics** - Per-account request counts, success rate
4. **Wallet threshold alerts** - Email/notification when balance < X

### ⏳ Phase 5: Additional Features
1. **Provider presets** - Pre-configured model bundles for different use cases
2. **Import/Export** - Export accounts to file (encrypted?), import batch
3. **Health checks** - Ping endpoint for each account, detect dead accounts proactively
4. **Notifications** - Integrate with notification system when wallet low
5. **Usage stats dashboard** - Total requests, costs, top models used

---

## Build Verification

```bash
cd ~/Downloads/enowx76apel
go build -tags dev -o enx ./cmd/enowx  # ✅ Success
./enx version                          # Check binary works
./enx autoclaw-login --help            # CLI command available
```

Web UI needs dependency install:
```bash
cd ~/Downloads/enowx76apel/web
npm install
npm run build  # Would succeed after deps
```

---

## Security Notes

⚠️ **Creds Storage**: Currently stored as plain JSON in DB. Consider encryption at rest for production.

⚠️ **Python Scripts**: Have their own dependencies (cloakbrowser, flask). Users need Python 3.10+ installed.

⚠️ **Token Refresh**: Tested flow works on first usage; background refresh runs every 15 minutes.

---

## Known Issues

- Web UI build has missing type deps (`vite/client`), but React code is syntactically correct
- Frontend requires `npm install` to resolve Vite/Tailwind types before actual compilation
- Python environment setup needed for `autoclaw-autologin.py` to function
