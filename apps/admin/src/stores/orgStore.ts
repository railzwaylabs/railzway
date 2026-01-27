import { create } from "zustand"

type Org = {
  id: string
  name: string
  role?: string
}

type OrgState = {
  orgs: Org[]
  currentOrg: Org | null
  testMode: boolean
  setTestMode: (enabled: boolean) => void
  setOrgs: (orgs: Org[]) => void
  setCurrentOrg: (org: Org | null) => void
}

export const useOrgStore = create<OrgState>((set) => ({
  orgs: [],
  currentOrg: null,
  testMode: false,
  setTestMode: (enabled) => set({ testMode: enabled }),
  setOrgs: (orgs) => set({ orgs }),
  setCurrentOrg: (org) => set({ currentOrg: org }),
}))
