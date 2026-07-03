import { useState } from "react";
import { X, ExternalLink } from "lucide-react";
import { ProviderIcon } from "./ProviderIcon";
import { autoclawApi } from "../lib/api";
import type { Provider } from "../lib/api";

type Tab = "manual" | "refresh";

const TABS: { id: Tab; label: string }[] = [
  { id: "manual", label: "Manual" },
  { id: "refresh", label: "Refresh token" },
];

export function AutoClawAddModal({
  provider,
  onClose,
  onSaved,
}: {
  provider: Provider;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [tab, setTab] = useState<Tab>("manual");

  return (
    <div
      className="absolute inset-0 z-50 flex items-center justify-center bg-black/50 p-4 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        className="flex max-h-[85%] w-full max-w-md flex-col overflow-hidden rounded-2xl border border-white/10 bg-[#11131a] shadow-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center gap-3 border-b border-white/5 px-4 py-3">
          <ProviderIcon icon={provider.icon} label={provider.label} size={32} />
          <div className="flex-1">
            <p className="text-sm font-semibold text-white">Add AutoClaw account</p>
            <p className="text-[11px] text-white/40">AutoGLM / Z.ai proxy account</p>
          </div>
          <button
            onClick={onClose}
            className="rounded-md p-1 text-white/40 hover:bg-white/10 hover:text-white"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="flex gap-1 border-b border-white/5 px-3 py-2">
          {TABS.map((t) => (
            <button
              key={t.id}
              onClick={() => setTab(t.id)}
              className={`rounded-md px-2.5 py-1 text-xs transition-colors ${
                tab === t.id
                  ? "bg-white/10 text-white"
                  : "text-white/50 hover:text-white/80"
              }`}
            >
              {t.label}
            </button>
          ))}
        </div>

        <div className="min-h-0 flex-1 overflow-auto p-4">
          {tab === "manual" && <ManualTab onSaved={onSaved} />}
          {tab === "refresh" && <RefreshTab onSaved={onSaved} />}
        </div>
      </div>
    </div>
  );
}

function Err({ msg }: { msg: string }) {
  if (!msg) return null;
  return (
    <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-xs text-red-300">
      {msg}
    </div>
  );
}

function PrimaryBtn({
  onClick,
  disabled,
  children,
}: {
  onClick: () => void;
  disabled?: boolean;
  children: React.ReactNode;
}) {
  return (
    <button
      onClick={onClick}
      disabled={disabled}
      className="w-full rounded-lg bg-white px-4 py-2 text-sm font-medium text-black transition-opacity hover:opacity-90 disabled:opacity-50"
    >
      {children}
    </button>
  );
}

function ManualTab({ onSaved }: { onSaved: () => void }) {
  const [text, setText] = useState("");
  const [err, setErr] = useState("");
  const [saving, setSaving] = useState(false);

  const format = () => {
    try {
      setText(JSON.stringify(JSON.parse(text), null, 2));
      setErr("");
    } catch {
      setErr("Not valid JSON yet");
    }
  };

  const submit = async () => {
    setErr("");
    setSaving(true);
    try {
      await api.autoclawManual(text);
      onSaved();
    } catch (e) {
      setErr(e instanceof Error ? e.message : "failed");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-3">
      <p className="text-xs text-white/50">
        Paste credentials JSON from the auto-login script. Keys: access_token, refresh_token, device_id, email, source_id.
      </p>
      <textarea
        value={text}
        onChange={(e) => setText(e.target.value)}
        onBlur={format}
        spellCheck={false}
        placeholder={
          '{\n  "access_token": "...",\n  "refresh_token": "...",\n  "email": "...",\n  "user_id": "..."\n}'
        }
        className="h-44 w-full resize-none rounded-lg border border-white/10 bg-black/30 p-3 font-mono text-xs text-white placeholder:text-white/25 focus:border-white/25 focus:outline-none"
      />
      <Err msg={err} />
      <div className="flex gap-2">
        <button
          onClick={format}
          className="rounded-lg border border-white/10 px-3 py-2 text-xs text-white/70 hover:bg-white/5"
        >
          Format
        </button>
        <PrimaryBtn onClick={submit} disabled={saving || !text.trim()}>
          {saving ? "Saving..." : "Add account"}
        </PrimaryBtn>
      </div>
    </div>
  );
}

function RefreshTab({ onSaved }: { onSaved: () => void }) {
  const [refreshToken, setRefreshToken] = useState("");
  const [sourceId, setSourceId] = useState("autoclaw");
  const [deviceId, setDeviceId] = useState("");
  const [label, setLabel] = useState("");
  const [err, setErr] = useState("");
  const [saving, setSaving] = useState(false);

  const submit = async () => {
    setErr("");
    setSaving(true);
    try {
      await api.autoclawRefresh({ refresh_token: refreshToken.trim(), source_id: sourceId.trim(), device_id: deviceId.trim(), label: label.trim() });
      onSaved();
    } catch (e) {
      setErr(e instanceof Error ? e.message : "failed");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-3">
      <p className="text-xs text-white/50">
        Paste a refresh token. The provider will exchange it for an access token on first use.
      </p>
      <Input label="Refresh token" value={refreshToken} onChange={setRefreshToken} secret />
      <Input label="Source ID (default: autoclaw)" value={sourceId} onChange={setSourceId} />
      <Input label="Device ID (optional)" value={deviceId} onChange={setDeviceId} />
      <Input label="Label (optional)" value={label} onChange={setLabel} />
      <Err msg={err} />
      <PrimaryBtn onClick={submit} disabled={saving || !refreshToken.trim()}>
        {saving ? "Saving..." : "Add account"}
      </PrimaryBtn>
    </div>
  );
}

function Input({
  label,
  value,
  onChange,
  secret,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  secret?: boolean;
}) {
  return (
    <label className="block">
      <span className="mb-1 block text-[11px] font-medium text-white/50">{label}</span>
      <input
        type={secret ? "password" : "text"}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        autoComplete="off"
        className="w-full rounded-lg border border-white/10 bg-white/5 px-3 py-2 text-sm text-white placeholder:text-white/25 focus:border-white/25 focus:outline-none"
      />
    </label>
  );
}
