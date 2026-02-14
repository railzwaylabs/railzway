import { useEffect, useState } from "react"
import { admin } from "@/api/client"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"
import { getErrorMessage } from "@/lib/api-errors"
import type { PaymentMethodConfig } from "../pages/OrgPaymentMethodConfigsPage"

interface PaymentMethodConfigDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  config: PaymentMethodConfig | null
  onSuccess: () => void
}

export function PaymentMethodConfigDialog({
  open,
  onOpenChange,
  config,
  onSuccess,
}: PaymentMethodConfigDialogProps) {
  const [isSaving, setIsSaving] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)

  // Form State
  const [methodType, setMethodType] = useState("card")
  const [methodName, setMethodName] = useState("")
  const [displayName, setDisplayName] = useState("")
  const [description, setDescription] = useState("")
  const [iconUrl, setIconUrl] = useState("")
  const [provider, setProvider] = useState("stripe")
  const [providerMethodType, setProviderMethodType] = useState("")
  const [priority, setPriority] = useState(0)
  const [isActive, setIsActive] = useState(true)
  const [availabilityRules, setAvailabilityRules] = useState("{}")

  useEffect(() => {
    if (open) {
      setFormError(null)
      if (config) {
        setMethodType(config.method_type)
        setMethodName(config.method_name)
        setDisplayName(config.display_name)
        setDescription(config.description || "")
        setIconUrl(config.icon_url || "")
        setProvider(config.provider)
        setProviderMethodType(config.provider_method_type || "")
        setPriority(config.priority)
        setIsActive(config.is_active)
        setAvailabilityRules(JSON.stringify(config.availability_rules, null, 2))
      } else {
        // Defaults
        setMethodType("card")
        setMethodName("")
        setDisplayName("")
        setDescription("")
        setIconUrl("")
        setProvider("stripe")
        setProviderMethodType("")
        setPriority(0)
        setIsActive(true)
        setAvailabilityRules(`{
  "countries": [],
  "currencies": []
}`)
      }
    }
  }, [open, config])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setIsSaving(true)
    setFormError(null)

    try {
      let parsedRules = {};
      try {
        parsedRules = JSON.parse(availabilityRules)
      } catch (e) {
        throw new Error("Availability Rules must be valid JSON")
      }

      const payload = {
        id: config?.id, // Optional for create, but used if we support UPSERT with ID from client (usually not)
        method_type: methodType,
        method_name: methodName,
        display_name: displayName,
        description: description || null,
        icon_url: iconUrl || null,
        provider,
        provider_method_type: providerMethodType || null,
        priority: Number(priority),
        availability_rules: parsedRules,
        is_active: isActive
      }

      // If editing, we use the specific ID update if needed, but our API uses POST /admin/payment-method-configs for upsert.
      // If config exists, we should probably pass the ID in the body to ensure it updates instead of creates if naming conflicts.
      if (config) {
        // @ts-ignore
        payload.id = config.id
      }

      await admin.post("/payment-method-configs", payload)
      onSuccess()
    } catch (err: any) {
      setFormError(getErrorMessage(err, "Unable to save configuration."))
    } finally {
      setIsSaving(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{config ? "Edit Checkout Option" : "Add Checkout Option"}</DialogTitle>
          <DialogDescription>
            Configure the routing and display properties for this payment method.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          {formError && (
            <Alert variant="destructive">
              <AlertDescription>{formError}</AlertDescription>
            </Alert>
          )}

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>Method Type</Label>
              <Select value={methodType} onValueChange={setMethodType}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="card">Card</SelectItem>
                  <SelectItem value="virtual_account">Virtual Account</SelectItem>
                  <SelectItem value="ewallet">E-Wallet</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="methodName">Method Name (Internal ID)</Label>
              <Input
                id="methodName"
                value={methodName}
                onChange={e => setMethodName(e.target.value)}
                placeholder="e.g. card_global_stripe"
                disabled={!!config} // Often internal IDs shouldn't change, but let's allow if backend handles it (unique constraint might fail)
              />
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="displayName">Display Name</Label>
            <Input
              id="displayName"
              value={displayName}
              onChange={e => setDisplayName(e.target.value)}
              placeholder="e.g. Credit Card"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="description">Description (Optional)</Label>
            <Input
              id="description"
              value={description}
              onChange={e => setDescription(e.target.value)}
              placeholder="e.g. Visa, Mastercard"
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>Provider</Label>
              <Select value={provider} onValueChange={setProvider}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="stripe">Stripe</SelectItem>
                  <SelectItem value="xendit">Xendit</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="providerMethodType">Provider Method Type (Optional)</Label>
              <Input
                id="providerMethodType"
                value={providerMethodType}
                onChange={e => setProviderMethodType(e.target.value)}
                placeholder="e.g. BCA, GOPAY (for Xendit)"
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="priority">Priority</Label>
              <Input
                id="priority"
                type="number"
                value={priority}
                onChange={e => setPriority(Number(e.target.value))}
              />
            </div>
            <div className="flex items-center space-x-2 pt-8">
              <Switch id="isActive" checked={isActive} onCheckedChange={setIsActive} />
              <Label htmlFor="isActive">Active</Label>
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="rules">Availability Rules (JSON)</Label>
            <Textarea
              id="rules"
              value={availabilityRules}
              onChange={e => setAvailabilityRules(e.target.value)}
              className="font-mono text-xs h-32"
            />
            <p className="text-xs text-text-muted">
              Example: {`{"countries": ["ID", "PH"], "currencies": ["IDR", "PHP"]}`}
            </p>
          </div>

          <DialogFooter>
            <Button type="button" variant="ghost" onClick={() => onOpenChange(false)}>Cancel</Button>
            <Button type="submit" disabled={isSaving}>Save</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
