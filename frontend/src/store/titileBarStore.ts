export const useTitleBarStore = defineStore("titileBar", {
  state: () => ({
    title: "Jarvist AI",
  }),
  actions: {
    setTitle(newTitle: string) {
      this.title = newTitle;
    },
  },
});
