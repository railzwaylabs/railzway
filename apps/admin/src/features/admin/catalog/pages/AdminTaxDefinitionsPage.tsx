import { useMemo, useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import { admin } from "@/api/client"
import { ForbiddenState } from "@/components/forbidden-state"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Spinner } from "@/components/ui/spinner"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Textarea } from "@/components/ui/textarea"
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

import { AdminCatalogTabs } from "@/features/admin/catalog/components/AdminCatalogTabs"
import { getErrorMessage, isForbiddenError } from "@/lib/api-errors"

type TaxDefinition = {
  id: string
  code: string
  name: string
  tax_mode: string
  rate?: number | null
  description?: string | null
  is_enabled: boolean
  is_rate_locked?: boolean
}

const fetchTaxDefinitions = async () => {
  const response = await admin.get("/tax-definitions")
  const payload = response.data?.data
  if (Array.isArray(payload)) {
    return payload as TaxDefinition[]
  }
  if (payload && typeof payload === "object") {
    const list = (payload as { tax_definitions?: TaxDefinition[] }).tax_definitions
    return Array.isArray(list) ? list : []
  }
  return []
}

const formatTaxMode = (value: string) => {
  const normalized = value.trim().toLowerCase()
  if (normalized === "exclusive") return "Exclusive"
  if (normalized === "inclusive") return "Inclusive"
  return value || "-"
}

const formatRate = (rate?: number | null) => {
  if (rate == null || Number.isNaN(rate)) return "Dynamic"
  const percent = rate * 100
  return new Intl.NumberFormat("en-US", {
    maximumFractionDigits: 4,
  }).format(percent) + "%"
}

const normalizeRate = (raw: string) => {
  const trimmed = raw.trim()
  if (!trimmed) return { value: null, error: null }
  const parsed = Number(trimmed)
  if (Number.isNaN(parsed) || parsed < 0) {
    return { value: null, error: "Rate must be a positive number." }
  }
  return { value: parsed / 100, error: null }
}

const rateToInput = (rate?: number | null) => {
  if (rate == null || Number.isNaN(rate)) return ""
  return (rate * 100).toString()
}

export default function AdminTaxDefinitionsPage() {
  const queryClient = useQueryClient()
  const { data, isLoading, error } = useQuery({
    queryKey: ["admin-tax-definitions"],
    queryFn: fetchTaxDefinitions,
  })

  const definitions = useMemo(() => data ?? [], [data])
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const [createName, setCreateName] = useState("")
  const [createCode, setCreateCode] = useState("")
  const [createMode, setCreateMode] = useState("exclusive")
  const [createRate, setCreateRate] = useState("")
  const [createDescription, setCreateDescription] = useState("")
  const [createError, setCreateError] = useState<string | null>(null)
  const [editingDefinition, setEditingDefinition] = useState<TaxDefinition | null>(null)
  const [editName, setEditName] = useState("")
  const [editMode, setEditMode] = useState("exclusive")
  const [editRate, setEditRate] = useState("")
  const [editDescription, setEditDescription] = useState("")
  const [editError, setEditError] = useState<string | null>(null)
  const [disableDefinition, setDisableDefinition] = useState<TaxDefinition | null>(null)
  const [disableError, setDisableError] = useState<string | null>(null)

  const createMutation = useMutation({
    mutationFn: async () => {
      const rate = normalizeRate(createRate)
      if (rate.error) {
        throw new Error(rate.error)
      }
      return admin.post("/tax-definitions", {
        name: createName.trim(),
        code: createCode.trim(),
        tax_mode: createMode,
        rate: rate.value,
        description: createDescription.trim() || null,
      })
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["admin-tax-definitions"] })
      setCreateName("")
      setCreateCode("")
      setCreateMode("exclusive")
      setCreateRate("")
      setCreateDescription("")
      setCreateError(null)
      setIsCreateOpen(false)
    },
    onError: (err: any) => {
      setCreateError(getErrorMessage(err, "Unable to create tax definition."))
    },
  })

  const updateMutation = useMutation({
    mutationFn: async (payload: {
      id: string
      name: string
      tax_mode: string
      rate: string
      description: string
    }) => {
      const rate = normalizeRate(payload.rate)
      if (rate.error) {
        throw new Error(rate.error)
      }
      return admin.patch(`/tax-definitions/${payload.id}`, {
        name: payload.name,
        tax_mode: payload.tax_mode,
        rate: rate.value,
        description: payload.description,
      })
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["admin-tax-definitions"] })
      setEditingDefinition(null)
      setEditError(null)
    },
    onError: (err: any) => {
      setEditError(getErrorMessage(err, "Unable to update tax definition."))
    },
  })

  const disableMutation = useMutation({
    mutationFn: async (id: string) => admin.post(`/tax-definitions/${id}/disable`),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["admin-tax-definitions"] })
      setDisableDefinition(null)
      setDisableError(null)
    },
    onError: (err: any) => {
      setDisableError(getErrorMessage(err, "Unable to disable tax definition."))
    },
  })

  if (isForbiddenError(error)) {
    return (
      <ForbiddenState description="You do not have access to tax definitions." />
    )
  }

  const isCreateDisabled =
    !createName.trim() || !createCode.trim() || createMutation.isPending

  return (
    <div className="space-y-6">
      <div className="space-y-3">
        <AdminCatalogTabs />
        <div className="space-y-1">
          <h1 className="text-2xl font-semibold">Tax definitions</h1>
          <p className="text-text-muted text-sm">
            Tax definitions determine how taxes are applied to customer invoices.
          </p>
          <p className="text-text-muted text-sm">
            Changes affect future invoices only. Finalized invoices are never modified.
          </p>
        </div>
      </div>

      <Card>
        <CardHeader>
          <div className="space-y-1">
            <CardTitle>Tax policies</CardTitle>
            <CardDescription>
              Tax definitions are snapshotted into invoices at finalization.
            </CardDescription>
          </div>
          <CardAction>
            <Dialog
              open={isCreateOpen}
              onOpenChange={(open) => {
                setIsCreateOpen(open)
                if (!open) {
                  setCreateError(null)
                }
              }}
            >
              <DialogTrigger asChild>
                <Button size="sm">Create tax definition</Button>
              </DialogTrigger>
              <DialogContent className="sm:max-w-xl">
                <DialogHeader>
                  <DialogTitle>Create tax definition</DialogTitle>
                  <DialogDescription>
                    Configure how tax should be applied for future invoices.
                  </DialogDescription>
                </DialogHeader>
                <form
                  className="space-y-4"
                  onSubmit={(event) => {
                    event.preventDefault()
                    setCreateError(null)
                    if (isCreateDisabled) return
                    createMutation.mutate()
                  }}
                >
                  {createError && (
                    <Alert variant="destructive">
                      <AlertTitle>Unable to save</AlertTitle>
                      <AlertDescription>{createError}</AlertDescription>
                    </Alert>
                  )}
                  <Alert>
                    <AlertTitle>Invoice snapshotting</AlertTitle>
                    <AlertDescription>
                      Tax definitions are snapshotted into invoices at finalization.
                    </AlertDescription>
                  </Alert>
                  <div className="grid gap-4 sm:grid-cols-2">
                    <div className="space-y-2">
                      <Label htmlFor="tax-name">Name</Label>
                      <Input
                        id="tax-name"
                        value={createName}
                        onChange={(event) => setCreateName(event.target.value)}
                        placeholder="EU VAT"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="tax-code">Code</Label>
                      <Input
                        id="tax-code"
                        value={createCode}
                        onChange={(event) => setCreateCode(event.target.value)}
                        placeholder="EU_VAT"
                      />
                    </div>
                  </div>
                  <div className="grid gap-4 sm:grid-cols-2">
                    <div className="space-y-2">
                      <Label htmlFor="tax-mode">Tax mode</Label>
                      <Select value={createMode} onValueChange={setCreateMode}>
                        <SelectTrigger id="tax-mode">
                          <SelectValue placeholder="Select tax mode" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="exclusive">Exclusive</SelectItem>
                          <SelectItem value="inclusive">Inclusive</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="tax-rate">Rate (%)</Label>
                      <Input
                        id="tax-rate"
                        type="number"
                        step="0.0001"
                        value={createRate}
                        onChange={(event) => setCreateRate(event.target.value)}
                        placeholder="Leave empty for dynamic"
                      />
                    </div>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="tax-description">Description</Label>
                    <Textarea
                      id="tax-description"
                      value={createDescription}
                      onChange={(event) => setCreateDescription(event.target.value)}
                      placeholder="Optional description for internal use."
                    />
                  </div>
                  <DialogFooter className="gap-2">
                    <Button
                      type="button"
                      variant="ghost"
                      onClick={() => setIsCreateOpen(false)}
                      disabled={createMutation.isPending}
                    >
                      Cancel
                    </Button>
                    <Button type="submit" disabled={isCreateDisabled}>
                      {createMutation.isPending ? "Saving..." : "Create definition"}
                    </Button>
                  </DialogFooter>
                </form>
              </DialogContent>
            </Dialog>
          </CardAction>
        </CardHeader>
        <CardContent>
          {isLoading && (
            <div className="flex items-center gap-2 text-text-muted text-sm">
              <Spinner />
              Loading tax definitions
            </div>
          )}
          {error && (
            <div className="text-status-error text-sm">
              {getErrorMessage(error, "Unable to load tax definitions.")}
            </div>
          )}
          {!isLoading && !error && definitions.length === 0 && (
            <Empty>
              <EmptyHeader>
                <EmptyMedia variant="icon">%</EmptyMedia>
                <EmptyTitle>No tax definitions yet</EmptyTitle>
                <EmptyDescription>
                  Create a tax definition to apply taxes during invoice finalization.
                </EmptyDescription>
              </EmptyHeader>
            </Empty>
          )}
          {!isLoading && !error && definitions.length > 0 && (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Tax name</TableHead>
                  <TableHead>Code</TableHead>
                  <TableHead>Rate</TableHead>
                  <TableHead>Tax mode</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {definitions.map((definition) => (
                  <TableRow key={definition.id}>
                    <TableCell className="font-medium">{definition.name}</TableCell>
                    <TableCell className="text-text-muted">{definition.code}</TableCell>
                    <TableCell className="text-text-muted">{formatRate(definition.rate)}</TableCell>
                    <TableCell>
                      <Badge variant="outline">{formatTaxMode(definition.tax_mode)}</Badge>
                    </TableCell>
                    <TableCell>
                      <Badge
                        variant={definition.is_enabled ? "secondary" : "outline"}
                        className={
                          definition.is_enabled
                            ? "border-status-success/30 bg-status-success/10 text-status-success"
                            : "text-text-muted"
                        }
                      >
                        {definition.is_enabled ? "Enabled" : "Disabled"}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex items-center justify-end gap-2">
                        <Button
                          size="sm"
                          variant="ghost"
                          onClick={() => {
                            setEditingDefinition(definition)
                            setEditName(definition.name ?? "")
                            setEditMode(definition.tax_mode ?? "exclusive")
                            setEditRate(rateToInput(definition.rate))
                            setEditDescription(definition.description ?? "")
                            setEditError(null)
                          }}
                        >
                          Edit
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          onClick={() => {
                            setDisableDefinition(definition)
                            setDisableError(null)
                          }}
                        >
                          Disable
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <Dialog
        open={Boolean(editingDefinition)}
        onOpenChange={(open) => {
          if (!open) {
            setEditingDefinition(null)
            setEditError(null)
          }
        }}
      >
        <DialogContent className="sm:max-w-xl">
          <DialogHeader>
            <DialogTitle>Edit tax definition</DialogTitle>
            <DialogDescription>
              Update name, mode, or rate. Code remains immutable.
            </DialogDescription>
          </DialogHeader>
          <form
            className="space-y-4"
            onSubmit={(event) => {
              event.preventDefault()
              if (!editingDefinition) return
              setEditError(null)
              updateMutation.mutate({
                id: editingDefinition.id,
                name: editName.trim(),
                tax_mode: editMode,
                rate: editRate,
                description: editDescription.trim(),
              })
            }}
          >
            {editError && (
              <Alert variant="destructive">
                <AlertTitle>Unable to save</AlertTitle>
                <AlertDescription>{editError}</AlertDescription>
              </Alert>
            )}
            <Alert>
              <AlertTitle>Invoice snapshotting</AlertTitle>
              <AlertDescription>
                Changes affect future invoices only. Finalized invoices are never modified.
              </AlertDescription>
            </Alert>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="tax-edit-name">Name</Label>
                <Input
                  id="tax-edit-name"
                  value={editName}
                  onChange={(event) => setEditName(event.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="tax-edit-code">Code</Label>
                <Input
                  id="tax-edit-code"
                  value={editingDefinition?.code ?? ""}
                  disabled
                />
              </div>
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="tax-edit-mode">Tax mode</Label>
                <Select value={editMode} onValueChange={setEditMode}>
                  <SelectTrigger id="tax-edit-mode">
                    <SelectValue placeholder="Select tax mode" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="exclusive">Exclusive</SelectItem>
                    <SelectItem value="inclusive">Inclusive</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label htmlFor="tax-edit-rate">Rate (%)</Label>
                <Input
                  id="tax-edit-rate"
                  type="number"
                  step="0.0001"
                  value={editRate}
                  onChange={(event) => setEditRate(event.target.value)}
                  placeholder="Leave empty for dynamic"
                  disabled={editingDefinition?.is_rate_locked === true}
                />
                {editingDefinition?.is_rate_locked && (
                  <p className="text-xs text-text-muted">
                    Rate changes are locked after a definition is used on finalized invoices.
                  </p>
                )}
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="tax-edit-description">Description</Label>
              <Textarea
                id="tax-edit-description"
                value={editDescription}
                onChange={(event) => setEditDescription(event.target.value)}
              />
            </div>
            <DialogFooter className="gap-2">
              <Button
                type="button"
                variant="ghost"
                onClick={() => setEditingDefinition(null)}
                disabled={updateMutation.isPending}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={updateMutation.isPending || !editName.trim()}>
                {updateMutation.isPending ? "Saving..." : "Save changes"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <AlertDialog
        open={Boolean(disableDefinition)}
        onOpenChange={(open) => {
          if (!open) {
            setDisableDefinition(null)
            setDisableError(null)
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Disable tax definition</AlertDialogTitle>
            <AlertDialogDescription>
              Disabling affects future invoices only. Past invoices remain unchanged.
            </AlertDialogDescription>
          </AlertDialogHeader>
          {disableError && (
            <Alert variant="destructive">
              <AlertTitle>Unable to disable</AlertTitle>
              <AlertDescription>{disableError}</AlertDescription>
            </Alert>
          )}
          <AlertDialogFooter>
            <AlertDialogCancel disabled={disableMutation.isPending}>
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              disabled={disableMutation.isPending || !disableDefinition}
              onClick={() => {
                if (!disableDefinition) return
                disableMutation.mutate(disableDefinition.id)
              }}
            >
              {disableMutation.isPending ? "Disabling..." : "Disable definition"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
