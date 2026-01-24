import { useEffect, useMemo, useState } from "react"
import { useLocation, useNavigate } from "react-router-dom"

import { auth } from "@/api/client"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { LoginForm } from "@/components/login-form"
import { useAuthStore } from "@/stores/authStore"
import logo from "@/assets/primary.svg"

type AuthProvider = {
  name: string
  display_name: string
  login_path: string
}

type AuthConfig = {
  local_login_enabled: boolean
  providers: AuthProvider[]
}

function ProviderButtons({ providers, redirectTo }: { providers: AuthProvider[]; redirectTo?: string }) {
  if (!providers.length) {
    return (
      <p className="text-center text-sm text-text-muted">
        No OAuth providers are configured for this instance.
      </p>
    )
  }

  return (
    <div className="space-y-3">
      {providers.map((provider) => (
        <Button asChild className="w-full" key={provider.name}>
          <a
            href={
              redirectTo
                ? `${provider.login_path}?redirectTo=${encodeURIComponent(redirectTo)}`
                : provider.login_path
            }
          >
            Sign in with {provider.display_name || provider.name}
          </a>
        </Button>
      ))}
    </div>
  )
}

function CloudLogin({ providers, redirectTo }: { providers: AuthProvider[]; redirectTo?: string }) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-bg-subtle/40 px-4 py-12">
      <Card className="w-full max-w-lg">
        <CardHeader className="items-center text-center">
          <img src={logo} className="mb-4 size-14" alt="Railzway logo" />
          <CardTitle className="text-2xl">Welcome to Railzway</CardTitle>
          <CardDescription>Sign in using your Railzway account.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <ProviderButtons providers={providers} redirectTo={redirectTo} />
          <p className="text-center text-sm text-text-muted">This instance only accepts OAuth sign-in.</p>
        </CardContent>
      </Card>
    </div>
  )
}

function LocalLogin({ providers, redirectTo }: { providers: AuthProvider[]; redirectTo?: string }) {
  const navigate = useNavigate()
  const login = useAuthStore((s) => s.login)
  const [error, setError] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(false)

  const handleSubmit = async (payload: { email: string; password: string }) => {
    setError(null)
    setIsLoading(true)
    try {
      await login(payload)
      const nextMustChangePassword = useAuthStore.getState().mustChangePassword
      const target = redirectTo || "/orgs"
      navigate(nextMustChangePassword ? "/change-password" : target, { replace: true })
    } catch (err: any) {
      setError(err?.message ?? "Unable to sign in.")
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-bg-subtle/40 px-4 py-12">
      <div className="w-full max-w-lg space-y-4">
        <LoginForm onSubmit={handleSubmit} isLoading={isLoading} error={error} showSignup={false}>
          {providers.length > 0 && (
            <div className="space-y-4">
              <p className="text-center text-sm text-text-muted">Or sign in with</p>
              <ProviderButtons providers={providers} redirectTo={redirectTo} />
            </div>
          )}
        </LoginForm>
        <p className="px-6 text-center text-sm text-text-muted">
          By clicking continue, you agree to our <a href="#">Terms of Service</a>{" "}
          and <a href="#">Privacy Policy</a>.
        </p>
      </div>
    </div>
  )
}

export default function LoginPage() {
  const [authConfig, setAuthConfig] = useState<AuthConfig | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)
  const location = useLocation()
  const redirectTo = useMemo(() => {
    const raw = new URLSearchParams(location.search).get("redirectTo")
    if (!raw) {
      return ""
    }
    const trimmed = raw.trim()
    if (!trimmed || !trimmed.startsWith("/") || trimmed.startsWith("//")) {
      return ""
    }
    return trimmed
  }, [location.search])

  useEffect(() => {
    let active = true
    setIsLoading(true)
    auth.get<AuthConfig>("/providers")
      .then((res) => {
        if (!active) return
        setAuthConfig(res.data)
        setLoadError(null)
      })
      .catch(() => {
        if (!active) return
        setLoadError("Unable to load authentication options. Please refresh.")
      })
      .finally(() => {
        if (!active) return
        setIsLoading(false)
      })
    return () => {
      active = false
    }
  }, [])

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center text-text-muted text-sm">
        Loading authentication...
      </div>
    )
  }

  if (loadError || !authConfig) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-bg-subtle/40 px-4 py-12">
        <Card className="w-full max-w-lg">
          <CardHeader className="text-center">
            <CardTitle className="text-xl">Sign-in unavailable</CardTitle>
            <CardDescription>{loadError ?? "Unable to load sign-in options."}</CardDescription>
          </CardHeader>
          <CardContent>
            <Button className="w-full" onClick={() => window.location.reload()}>
              Retry
            </Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (!authConfig.local_login_enabled) {
    return <CloudLogin providers={authConfig.providers ?? []} redirectTo={redirectTo} />
  }

  return <LocalLogin providers={authConfig.providers ?? []} redirectTo={redirectTo} />
}
