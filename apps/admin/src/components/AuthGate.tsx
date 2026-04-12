import { ReactNode, useEffect, useMemo, useState } from "react";

import { api } from "../lib/api";
import { clearOrgId, clearToken, isAuthRequired, setOrgId, setToken } from "../lib/auth";

type AuthState = "checking" | "login" | "change" | "ready";

type AuthGateProps = {
  children: ReactNode;
};

export default function AuthGate({ children }: AuthGateProps) {
  const authRequired = useMemo(() => isAuthRequired(), []);
  const [state, setState] = useState<AuthState>(authRequired ? "checking" : "ready");
  const [error, setError] = useState<string>("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!authRequired) {
      setState("ready");
      return;
    }
    api.auth
      .me()
      .then((res) => {
        if (res.orgId) {
          setOrgId(res.orgId);
        }
        if (res.mustChangePassword) {
          setState("change");
        } else {
          setState("ready");
        }
      })
      .catch(() => {
        clearOrgId();
        setState("login");
      });
  }, [authRequired]);

  const handleLogin = async (email: string, password: string) => {
    setLoading(true);
    setError("");
    try {
      const resp = await api.auth.login({ email, password });
      setToken(resp.token);
      if (resp.orgId) {
        setOrgId(resp.orgId);
      }
      setState(resp.mustChangePassword ? "change" : "ready");
    } catch (err) {
      const message = err instanceof Error ? err.message : "Login failed";
      setError(message);
      clearToken();
      clearOrgId();
      setState("login");
    } finally {
      setLoading(false);
    }
  };

  const handleSkipPassword = async () => {
    setLoading(true);
    setError("");
    try {
      await api.auth.skipPasswordChange();
      setState("ready");
    } catch (err) {
      const message = err instanceof Error ? err.message : "Skip failed";
      setError(message);
    } finally {
      setLoading(false);
    }
  };

  const handleChangePassword = async (currentPassword: string, newPassword: string) => {
    setLoading(true);
    setError("");
    try {
      await api.auth.changePassword({ currentPassword, newPassword });
      setState("ready");
    } catch (err) {
      const message = err instanceof Error ? err.message : "Password update failed";
      setError(message);
    } finally {
      setLoading(false);
    }
  };

  if (state === "ready") {
    return <>{children}</>;
  }

  if (state === "change") {
    return (
      <AuthShell
        title="Change Password"
        subtitle="Please update your password to continue."
        error={error}
      >
        <ChangePasswordForm onSubmit={handleChangePassword} onSkip={handleSkipPassword} loading={loading} />
      </AuthShell>
    );
  }

  if (state === "login") {
    return (
      <AuthShell title="Admin Login" subtitle="Sign in to access the Railzway console." error={error}>
        <LoginForm onSubmit={handleLogin} loading={loading} />
      </AuthShell>
    );
  }

  return (
    <div className="auth-shell">
      <div className="auth-card">
        <div className="auth-title">Checking session…</div>
        <div className="auth-subtitle">Hold on while we verify your access.</div>
        <div className="auth-loader" />
      </div>
    </div>
  );
}

type AuthShellProps = {
  title: string;
  subtitle: string;
  error?: string;
  children: ReactNode;
};

function AuthShell({ title, subtitle, error, children }: AuthShellProps) {
  return (
    <div className="auth-shell">
      <div className="auth-card">
        <div className="auth-title">{title}</div>
        <div className="auth-subtitle">{subtitle}</div>
        {error ? <div className="auth-error">{error}</div> : null}
        {children}
      </div>
    </div>
  );
}

type LoginFormProps = {
  onSubmit: (email: string, password: string) => void;
  loading: boolean;
};

function LoginForm({ onSubmit, loading }: LoginFormProps) {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");

  return (
    <form
      className="auth-form"
      onSubmit={(event) => {
        event.preventDefault();
        onSubmit(email.trim(), password);
      }}
    >
      <label className="auth-label">
        Email
        <input
          type="email"
          className="auth-input"
          value={email}
          onChange={(event) => setEmail(event.target.value)}
          placeholder="admin@railzway.com"
          required
          data-testid="auth-email"
        />
      </label>
      <label className="auth-label">
        Password
        <input
          type="password"
          className="auth-input"
          value={password}
          onChange={(event) => setPassword(event.target.value)}
          placeholder="Enter password"
          required
          data-testid="auth-password"
        />
      </label>
      <button className="auth-button" type="submit" disabled={loading} data-testid="auth-submit">
        {loading ? "Signing in…" : "Sign in"}
      </button>
    </form>
  );
}

type ChangePasswordFormProps = {
  onSubmit: (currentPassword: string, newPassword: string) => void;
  onSkip: () => void;
  loading: boolean;
};

function ChangePasswordForm({ onSubmit, onSkip, loading }: ChangePasswordFormProps) {
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");

  return (
    <form
      className="auth-form"
      onSubmit={(event) => {
        event.preventDefault();
        onSubmit(currentPassword, newPassword);
      }}
    >
      <label className="auth-label">
        Current password
        <input
          type="password"
          className="auth-input"
          value={currentPassword}
          onChange={(event) => setCurrentPassword(event.target.value)}
          placeholder="Current password"
          required
          data-testid="auth-current-password"
        />
      </label>
      <label className="auth-label">
        New password
        <input
          type="password"
          className="auth-input"
          value={newPassword}
          onChange={(event) => setNewPassword(event.target.value)}
          placeholder="New password"
          minLength={8}
          required
          data-testid="auth-new-password"
        />
      </label>
      <button className="auth-button" type="submit" disabled={loading} data-testid="auth-change-submit">
        {loading ? "Updating…" : "Update password"}
      </button>
      <button className="auth-link" type="button" onClick={onSkip} disabled={loading} data-testid="auth-skip">
        Skip for now
      </button>
    </form>
  );
}
