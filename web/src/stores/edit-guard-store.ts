import { create } from "zustand";

type EditGuardState = {
  lockCount: number;
  isEditing: boolean;
  beginEdit: () => void;
  endEdit: () => void;
};

export const useEditGuardStore = create<EditGuardState>((set) => ({
  lockCount: 0,
  isEditing: false,
  beginEdit: () =>
    set((state) => {
      const lockCount = state.lockCount + 1;
      return { lockCount, isEditing: lockCount > 0 };
    }),
  endEdit: () =>
    set((state) => {
      const lockCount = Math.max(0, state.lockCount - 1);
      return { lockCount, isEditing: lockCount > 0 };
    }),
}));
