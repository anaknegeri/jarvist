import "./assets/index.css";

import { createApp } from "vue";
import VueKonva from "vue-konva";
import App from "./App.vue";
import { initializeApp, setupAppCleanup } from "./appSetup";
import router from "./router";

const app = createApp(App);
const pinia = createPinia();

app.use(pinia);
app.use(router);
app.use(VueKonva);

initializeApp();
setupAppCleanup();

app.mount("#app");
