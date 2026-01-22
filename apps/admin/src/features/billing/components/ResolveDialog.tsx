import { useState, useEffect } from "react"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import type { EntityType, ResolutionType } from "../types/ia-types"

interface ResolveDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSubmit: (resolution: string, notes?: string) => void
  entityType: EntityType
  entityName: string
}

const RESOLUTION_OPTIONS: { value: ResolutionType; label: string; description: string }[] = [
  {
    value: "payment_received",
    label: "Payment Received",
    description: "Customer has paid the outstanding amount",
  },
  {
    value: "issue_fixed",
    label: "Issue Fixed",
    description: "The billing or payment issue has been resolved",
  },
  {
    value: "customer_contacted",
    label: "Customer Contacted",
    description: "Successfully reached out to customer",
  },
  {
    value: "escalated_to_manager",
    label: "Escalated to Manager",
    description: "Requires manager intervention",
  },
  {
    value: "other",
    label: "Other",
    description: "Custom resolution (provide details in notes)",
  },
]

export function ResolveDialog({
  open,
  onOpenChange,
  onSubmit,
  entityType,
  entityName,
}: ResolveDialogProps) {
  const [resolution, setResolution] = useState<ResolutionType | "">("")
  const [notes, setNotes] = useState("")

  // Reset form when dialog opens
  useEffect(() => {
    if (open) {
      setResolution("")
      setNotes("")
    }
  }, [open])

  const handleSubmit = () => {
    if (!resolution) return

    // Combine resolution type with notes if provided
    const fullResolution = notes.trim()
      ? `${resolution}: ${notes.trim()}`
      : resolution

    onSubmit(fullResolution, notes.trim() || undefined)
    onOpenChange(false)
  }

  const selectedOption = RESOLUTION_OPTIONS.find((opt) => opt.value === resolution)

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>Resolve Assignment</DialogTitle>
          <DialogDescription>
            Mark this {entityType} ({entityName}) as resolved. This will move it to your Recently
            Resolved list.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-6 py-4">
          <div className="grid gap-3">
            <Label htmlFor="resolution">Resolution Type</Label>
            <Select value={resolution} onValueChange={(value) => setResolution(value as ResolutionType)}>
              <SelectTrigger id="resolution">
                <SelectValue placeholder="Select resolution type..." />
              </SelectTrigger>
              <SelectContent>
                {RESOLUTION_OPTIONS.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    <div className="flex flex-col items-start">
                      <span className="font-medium">{option.label}</span>
                      <span className="text-xs text-muted-foreground">{option.description}</span>
                    </div>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {selectedOption && (
              <p className="text-xs text-muted-foreground">{selectedOption.description}</p>
            )}
          </div>

          <div className="grid gap-3">
            <Label htmlFor="notes">
              Notes {resolution === "other" && <span className="text-destructive">*</span>}
            </Label>
            <Textarea
              id="notes"
              placeholder={
                resolution === "other"
                  ? "Please provide details about the resolution..."
                  : "Optional: Add any additional context or notes..."
              }
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              rows={4}
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={!resolution || (resolution === "other" && !notes.trim())}
          >
            Resolve
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
