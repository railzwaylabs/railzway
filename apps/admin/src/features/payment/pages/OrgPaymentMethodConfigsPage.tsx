import { useCallback, useEffect, useState } from "react"
import { useParams } from "react-router-dom"
import { Plus } from "lucide-react"

import { admin } from "@/api/client"
import { ForbiddenState } from "@/components/forbidden-state"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { getErrorMessage, isForbiddenError } from "@/lib/api-errors"
import { MoreHorizontal } from "lucide-react"
import { PaymentMethodConfigDialog } from "../components/PaymentMethodConfigDialog"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"

export type PaymentMethodConfig = {
  id: string
  org_id: string
  method_type: string
  method_name: string
  display_name: string
  description?: string
  icon_url?: string
  priority: number
  availability_rules: Record<string, any>
  provider: string
  provider_method_type?: string
  is_active: boolean
  created_at: string
  updated_at: string
}

export default function OrgPaymentMethodConfigsPage() {
  const { orgId } = useParams()
  const [configs, setConfigs] = useState<PaymentMethodConfig[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [isForbidden, setIsForbidden] = useState(false)

  const [activeConfig, setActiveConfig] = useState<PaymentMethodConfig | null>(null)
  const [isDialogOpen, setIsDialogOpen] = useState(false)

  const [configToDelete, setConfigToDelete] = useState<PaymentMethodConfig | null>(null)
  const [toggleConfig, setToggleConfig] = useState<{ config: PaymentMethodConfig, nextValue: boolean } | null>(null)

  const loadData = useCallback(async () => {
    if (!orgId) {
      setIsLoading(false)
      return
    }

    setIsLoading(true)
    setLoadError(null)
    setIsForbidden(false)

    try {
      const res = await admin.get<{ configs: PaymentMethodConfig[] }>("/payment-method-configs")
      setConfigs(Array.isArray(res.data?.configs) ? res.data.configs : [])
    } catch (err: any) {
      if (isForbiddenError(err)) {
        setIsForbidden(true)
      } else {
        setLoadError(getErrorMessage(err, "Unable to load payment method configurations."))
      }
      setConfigs([])
    } finally {
      setIsLoading(false)
    }
  }, [orgId])

  useEffect(() => {
    void loadData()
  }, [loadData])

  const handleEdit = (config: PaymentMethodConfig) => {
    setActiveConfig(config)
    setIsDialogOpen(true)
  }

  const handleCreate = () => {
    setActiveConfig(null)
    setIsDialogOpen(true)
  }

  const handleDelete = async () => {
    if (!configToDelete) return
    try {
      await admin.delete(`/payment-method-configs/${configToDelete.id}`)
      await loadData()
    } catch (err: any) {
      setLoadError(getErrorMessage(err, "Unable to delete configuration."))
    } finally {
      setConfigToDelete(null)
    }
  }

  const handleToggle = async () => {
    if (!toggleConfig) return
    try {
      await admin.post(`/payment-method-configs/${toggleConfig.config.id}/toggle`, { is_active: toggleConfig.nextValue })
      await loadData()
    } catch (err: any) {
      setLoadError(getErrorMessage(err, "Unable to toggle status."))
    } finally {
      setToggleConfig(null)
    }
  }

  const onScanComplete = () => {
    setIsDialogOpen(false)
    loadData()
  }

  if (isForbidden) {
    return <ForbiddenState description="You do not have access to payment method configurations." />
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="space-y-1">
          <h1 className="text-2xl font-semibold">Checkout Options</h1>
          <p className="text-sm text-text-muted">
            Configure how payment methods are routed and displayed to customers.
          </p>
        </div>
        <Button onClick={handleCreate}>
          <Plus className="mr-2 h-4 w-4" />
          Add Method
        </Button>
      </div>

      {loadError && (
        <Alert variant="destructive">
          <AlertDescription>{loadError}</AlertDescription>
        </Alert>
      )}

      <Card>
        <CardHeader>
          <CardTitle>Routing Rules</CardTitle>
          <CardDescription>
            Define availability rules and priorities for each payment method.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading && <div className="text-sm text-text-muted">Loading configurations...</div>}
          {!isLoading && configs.length === 0 && (
            <div className="text-sm text-text-muted">No payment methods configured.</div>
          )}
          {!isLoading && configs.length > 0 && (
            <div className="overflow-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Display Name</TableHead>
                    <TableHead>Method Name</TableHead>
                    <TableHead>Provider</TableHead>
                    <TableHead>Type</TableHead>
                    <TableHead>Priority</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead className="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {configs.map((config) => (
                    <TableRow key={config.id}>
                      <TableCell className="font-medium">
                        {config.display_name}
                        {config.description && <div className="text-xs text-text-muted truncate max-w-[200px]">{config.description}</div>}
                      </TableCell>
                      <TableCell>
                        <code className="bg-background-subtle px-1 py-0.5 rounded text-xs">{config.method_name}</code>
                      </TableCell>
                      <TableCell className="capitalize">{config.provider}</TableCell>
                      <TableCell className="capitalize">{config.method_type.replace('_', ' ')}</TableCell>
                      <TableCell>{config.priority}</TableCell>
                      <TableCell>
                        <Badge variant={config.is_active ? "default" : "secondary"}>
                          {config.is_active ? "Active" : "Inactive"}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-right">
                        <DropdownMenu>
                          <DropdownMenuTrigger asChild>
                            <Button variant="ghost" size="icon">
                              <MoreHorizontal className="h-4 w-4" />
                            </Button>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent align="end">
                            <DropdownMenuItem onClick={() => handleEdit(config)}>Edit</DropdownMenuItem>
                            <DropdownMenuItem onClick={() => setToggleConfig({ config, nextValue: !config.is_active })}>
                              {config.is_active ? "Disable" : "Enable"}
                            </DropdownMenuItem>
                            <DropdownMenuSeparator />
                            <DropdownMenuItem className="text-status-error" onClick={() => setConfigToDelete(config)}>
                              Delete
                            </DropdownMenuItem>
                          </DropdownMenuContent>
                        </DropdownMenu>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>

      <PaymentMethodConfigDialog
        open={isDialogOpen}
        onOpenChange={setIsDialogOpen}
        config={activeConfig}
        onSuccess={onScanComplete}
      />

      <AlertDialog open={!!configToDelete} onOpenChange={(open) => !open && setConfigToDelete(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete payment method?</AlertDialogTitle>
            <AlertDialogDescription>
              This will remove the routing rule "{configToDelete?.method_name}". This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={handleDelete} className="bg-status-error hover:bg-status-error-hover text-white">
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={!!toggleConfig} onOpenChange={(open) => !open && setToggleConfig(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{toggleConfig?.nextValue ? "Enable" : "Disable"} payment method?</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to {toggleConfig?.nextValue ? "enable" : "disable"} "{toggleConfig?.config.method_name}"?
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={handleToggle}>
              Confirm
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
