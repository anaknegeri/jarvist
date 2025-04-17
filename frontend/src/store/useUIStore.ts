export const useUIStore = defineStore("ui", {
  state: () => ({
    cameraViewPreference:
      localStorage.getItem("cameraViewPreference") || "grid",
  }),
  actions: {
    setCameraView(view: string) {
      this.cameraViewPreference = view;
      localStorage.setItem("cameraViewPreference", view);
    },
  },
});
