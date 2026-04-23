import { useCallback, useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import PageHeader from "../components/PageHeader";
import { toast } from "../components/Toast";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { api } from "../lib/api";
import { formatDateTime } from "../lib/display";
import { useOrgPath } from "../lib/org";
import type { Customer, TestClock } from "../lib/types";

function IconClock() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5">
      <circle cx="10" cy="10" r="7" />
      <path d="M10 6v4l3 2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

function toDateTimeLocal(value: string | Date) {
  const date = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "";
  }
  const offsetMs = date.getTimezoneOffset() * 60_000;
  return new Date(date.getTime() - offsetMs).toISOString().slice(0, 16);
}

function epochSecondsFromLocal(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return null;
  }
  return Math.floor(date.getTime() / 1000);
}

export default function TestClockPage() {
  const orgPath = useOrgPath();
  const [clocks, setClocks] = useState<TestClock[]>([]);
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [selectedID, setSelectedID] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [createForm, setCreateForm] = useState({
    name: "",
    frozenTime: toDateTimeLocal(new Date())
  });
  const [advanceTime, setAdvanceTime] = useState(toDateTimeLocal(new Date()));

  const selectedClock = useMemo(
    () => clocks.find((clock) => clock.id === selectedID) ?? clocks[0] ?? null,
    [clocks, selectedID]
  );

  const attachedCustomers = useMemo(() => {
    if (!selectedClock) {
      return [];
    }
    return customers.filter((customer) => customer.test_clock_id === selectedClock.id);
  }, [customers, selectedClock]);

  const load = useCallback(async () => {
    try {
      setLoading(true);
      const [clockResp, customerResp] = await Promise.all([
        api.testClock.list(),
        api.customers.list({ page_size: 100 })
      ]);
      setClocks(clockResp.test_clocks);
      setCustomers(customerResp.customers);
      setSelectedID((current) => current || clockResp.test_clocks[0]?.id || "");
      const currentClock = clockResp.test_clocks[0];
      if (currentClock) {
        setAdvanceTime(toDateTimeLocal(currentClock.current_time));
      }
    } catch (err) {
      toast.error("Unable to load test clocks", err instanceof Error ? err.message : undefined);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    if (selectedClock) {
      setAdvanceTime(toDateTimeLocal(selectedClock.current_time));
    }
  }, [selectedClock?.id]);

  const handleCreate = useCallback(async () => {
    const frozenTime = epochSecondsFromLocal(createForm.frozenTime);
    if (!frozenTime) {
      toast.error("Choose a valid starting time.");
      return;
    }
    try {
      setSaving(true);
      const clock = await api.testClock.upsert({
        frozen_time: frozenTime,
        name: createForm.name.trim() || "Test clock",
        status: "active"
      });
      setClocks((prev) => [clock, ...prev.filter((item) => item.id !== clock.id)]);
      setSelectedID(clock.id);
      setAdvanceTime(toDateTimeLocal(clock.current_time));
      setCreateForm({ name: "", frozenTime: toDateTimeLocal(new Date()) });
      toast.success("Test clock created", clock.name);
    } catch (err) {
      toast.error("Unable to create test clock", err instanceof Error ? err.message : undefined);
    } finally {
      setSaving(false);
    }
  }, [createForm]);

  const handleAdvance = useCallback(async () => {
    if (!selectedClock) {
      toast.error("Select a test clock first.");
      return;
    }
    const frozenTime = epochSecondsFromLocal(advanceTime);
    if (!frozenTime) {
      toast.error("Choose a valid target time.");
      return;
    }
    try {
      setSaving(true);
      const clock = await api.testClock.advance(selectedClock.id, { frozen_time: frozenTime });
      setClocks((prev) => prev.map((item) => (item.id === clock.id ? clock : item)));
      setAdvanceTime(toDateTimeLocal(clock.current_time));
      toast.success("Test clock advanced", formatDateTime(clock.current_time));
      void load();
    } catch (err) {
      toast.error("Unable to advance test clock", err instanceof Error ? err.message : undefined);
    } finally {
      setSaving(false);
    }
  }, [advanceTime, load, selectedClock]);

  const handleStatus = useCallback(async (status: "paused" | "active") => {
    if (!selectedClock) {
      return;
    }
    try {
      setSaving(true);
      const clock = status === "paused"
        ? await api.testClock.pause(selectedClock.id)
        : await api.testClock.resume(selectedClock.id);
      setClocks((prev) => prev.map((item) => (item.id === clock.id ? clock : item)));
      toast.success(status === "paused" ? "Test clock paused" : "Test clock resumed");
    } catch (err) {
      toast.error("Unable to update test clock", err instanceof Error ? err.message : undefined);
    } finally {
      setSaving(false);
    }
  }, [selectedClock]);

  return (
    <div className="page-content">
      <PageHeader
        icon={<IconClock />}
        title="Test clocks"
        description="Create isolated clocks, attach customers, and advance simulated billing timelines."
        actions={(
          <Button variant="outline" asChild data-testid="testclock-audit-logs">
            <Link to={orgPath("/audit-logs")}>Audit logs</Link>
          </Button>
        )}
      />

      <div className="panel">
        <div className="action-section" style={{ border: "none" }}>
          <div className="action-section-title">Create test clock</div>
          <div className="action-fields">
            <div className="action-field">
              <Label className="action-label">Name</Label>
              <Input
                className="action-input"
                value={createForm.name}
                onChange={(event) => setCreateForm((prev) => ({ ...prev, name: event.target.value }))}
                placeholder="Subscription auto-renew simulation"
                data-testid="testclock-name"
              />
            </div>
            <div className="action-field">
              <Label className="action-label">Frozen time</Label>
              <Input
                className="action-input"
                type="datetime-local"
                value={createForm.frozenTime}
                onChange={(event) => setCreateForm((prev) => ({ ...prev, frozenTime: event.target.value }))}
                data-testid="testclock-current-time"
              />
            </div>
          </div>
          <div className="action-buttons">
            <Button onClick={handleCreate} disabled={saving} data-testid="testclock-save">
              {saving ? "Saving..." : "Create clock"}
            </Button>
          </div>
        </div>

        <div className="action-section">
          <div className="action-section-title">Active clocks</div>
          {loading ? (
            <div className="muted">Loading...</div>
          ) : clocks.length === 0 ? (
            <div className="muted">No test clocks yet.</div>
          ) : (
            <div className="table-wrapper">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>Name</th>
                    <th>Status</th>
                    <th>Frozen time</th>
                    <th>Customers</th>
                    <th />
                  </tr>
                </thead>
                <tbody>
                  {clocks.map((clock) => (
                    <tr key={clock.id}>
                      <td>{clock.name || "Test clock"}</td>
                      <td><span className="status-badge">{clock.status}</span></td>
                      <td>{formatDateTime(clock.current_time)}</td>
                      <td>{customers.filter((customer) => customer.test_clock_id === clock.id).length}</td>
                      <td>
                        <Button size="sm" variant={selectedClock?.id === clock.id ? "default" : "secondary"} onClick={() => setSelectedID(clock.id)}>
                          Select
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>

        {selectedClock ? (
          <div className="action-section">
            <div className="action-section-title">Advance selected clock</div>
            <div className="flag-toolbar">
              <div>
                <div className="muted">Selected</div>
                <strong>{selectedClock.name || selectedClock.id}</strong>
              </div>
              <div>
                <div className="muted">Current frozen time</div>
                <strong>{formatDateTime(selectedClock.current_time)}</strong>
              </div>
              <div>
                <div className="muted">Attached customers</div>
                <strong>{attachedCustomers.length}</strong>
              </div>
            </div>
            <div className="action-fields">
              <div className="action-field">
                <Label className="action-label">Target frozen time</Label>
                <Input
                  className="action-input"
                  type="datetime-local"
                  value={advanceTime}
                  onChange={(event) => setAdvanceTime(event.target.value)}
                  data-testid="testclock-advance-seconds"
                />
              </div>
            </div>
            <div className="action-buttons">
              <Button variant="secondary" disabled={saving} onClick={handleAdvance} data-testid="testclock-advance">
                {saving ? "Advancing..." : "Advance"}
              </Button>
              <Button variant="outline" disabled={saving} onClick={() => handleStatus("paused")} data-testid="testclock-pause">
                Pause
              </Button>
              <Button variant="outline" disabled={saving} onClick={() => handleStatus("active")} data-testid="testclock-resume">
                Resume
              </Button>
            </div>
            <div className="action-section" style={{ paddingLeft: 0, paddingRight: 0 }}>
              <div className="action-section-title">Attached customers</div>
              {attachedCustomers.length === 0 ? (
                <div className="muted">No customers are attached to this clock.</div>
              ) : (
                <div className="table-wrapper">
                  <table className="data-table">
                    <tbody>
                      {attachedCustomers.map((customer) => (
                        <tr key={customer.id}>
                          <td>{customer.name}</td>
                          <td>{customer.email}</td>
                          <td><Link to={orgPath(`/customers/${customer.id}/edit`)}>Edit</Link></td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          </div>
        ) : null}
      </div>
    </div>
  );
}
