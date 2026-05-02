import {
  Component,
  JSX,
  createContext,
  useContext
} from "solid-js";
import { createStore, produce } from "solid-js/store";

export interface Tab {
  id: string;
  label: string;
}

interface TabsState {
  tabs: Tab[];
  activeTabId: string;
}

interface TabsContextValue {
  state: TabsState;
  activeTabId: () => string;
  tabs: () => Tab[];
  addTab: () => string;
  closeTab: (tabId: string) => void;
  setActiveTab: (tabId: string) => void;
  renameTab: (tabId: string, label: string) => void;
}

const TabsContext = createContext<TabsContextValue>();

let nextId = 1;

function generateTabId(): string {
  return `tab-${nextId++}`;
}

export const TabsProvider: Component<{ children: JSX.Element }> = (props) => {
  const initialId = generateTabId();

  const [state, setState] = createStore<TabsState>({
    tabs: [{ id: initialId, label: "Terminal 1" }],
    activeTabId: initialId
  });

  const activeTabId = (): string => state.activeTabId;
  const tabs = (): Tab[] => state.tabs;

  const addTab = (): string => {
    const id = generateTabId();
    const num = state.tabs.length + 1;
    setState(
      produce((s) => {
        s.tabs.push({ id, label: `Terminal ${num}` });
        s.activeTabId = id;
      })
    );
    return id;
  };

  const closeTab = (tabId: string): void => {
    if (state.tabs.length <= 1) return;

    setState(
      produce((s) => {
        const idx = s.tabs.findIndex((t) => t.id === tabId);
        if (idx === -1) return;

        s.tabs.splice(idx, 1);

        if (s.activeTabId === tabId) {
          const newIdx = Math.min(idx, s.tabs.length - 1);
          const nextTab = s.tabs[newIdx];
          if (nextTab) {
            s.activeTabId = nextTab.id;
          }
        }
      })
    );
  };

  const setActiveTab = (tabId: string): void => {
    setState("activeTabId", tabId);
  };

  const renameTab = (tabId: string, label: string): void => {
    setState(
      produce((s) => {
        const tab = s.tabs.find((t) => t.id === tabId);
        if (tab) {
          tab.label = label;
        }
      })
    );
  };

  return (
    <TabsContext.Provider
      value={{
        state,
        activeTabId,
        tabs,
        addTab,
        closeTab,
        setActiveTab,
        renameTab
      }}
    >
      {props.children}
    </TabsContext.Provider>
  );
};

export function useTabs(): TabsContextValue {
  const ctx = useContext(TabsContext);
  if (!ctx) {
    throw new Error("useTabs must be used within TabsProvider");
  }
  return ctx;
}
