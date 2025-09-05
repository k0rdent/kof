import { StoreApi, UseBoundStore, create } from "zustand";

export interface Response<T> {
  items: Record<string, T>;
}

export interface ProviderStore<TItems, TItem> {
  items: TItems | null;
  selectedItem: TItem | null;
  isLoading: boolean;
  error?: Error;
  fetch: () => Promise<void>;
  selectItem: (name: string) => void;
}

export abstract class Provider<TItems, TItem, TRaw> {
  private readonly _endpoint: string;
  private readonly _store: UseBoundStore<
    StoreApi<ProviderStore<TItems, TItem>>
  >;

  constructor(endpoint: string) {
    this._endpoint = endpoint;
    this._store = create<ProviderStore<TItems, TItem>>((set, get) => ({
      items: null,
      selectedItem: null,
      isLoading: false,
      error: undefined,
      fetch: async () => {
        try {
          set({ isLoading: true, error: undefined });
          const resp = await this.fetch();
          const items = this.buildItems(resp);
          set({ items, isLoading: false });
        } catch (e) {
          set({ items: null, isLoading: false, error: e as Error });
        }
      },
      selectItem: (name: string) => {
        const items = get().items;
        if (!items) return;
        const selected = this.selectFrom(items, name);
        set({ selectedItem: selected ?? null });
      },
    }));

    this._store.getState().fetch();
  }

  public get use(): UseBoundStore<StoreApi<ProviderStore<TItems, TItem>>> {
    return this._store;
  }

  private async fetch(): Promise<Record<string, TRaw>> {
    const res = await fetch(this._endpoint, { method: "GET" });
    if (!res.ok) {
      throw new Error(`Response status ${res.status}`);
    }
    const json = (await res.json()) as Response<TRaw>;
    return json.items;
  }

  protected abstract buildItems(rawItems: Record<string, TRaw>): TItems;

  protected abstract selectFrom(items: TItems, name: string): TItem | null;
}
