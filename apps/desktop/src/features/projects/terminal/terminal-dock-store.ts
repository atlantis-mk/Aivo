import { create } from "zustand";

type TerminalDockState = {
  open: boolean;
  setOpen: (open: boolean) => void;
  state: "expanded" | "collapsed";
  toggleDock: () => void;
};

export const useTerminalDock = create<TerminalDockState>((set, get) => ({
  open: false,
  setOpen: (open) =>
    set({
      open,
      state: open ? "expanded" : "collapsed",
    }),
  state: "collapsed",
  toggleDock: () => get().setOpen(!get().open),
}));
