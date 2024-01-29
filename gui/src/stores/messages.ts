import { ref, computed, watch, getCurrentInstance } from "vue";
import type { Ref } from "vue";
import { defineStore } from "pinia";
import dayjs from "dayjs";
import { Centrifuge } from "centrifuge";
import utc from "dayjs/plugin/utc";
import timezone from "dayjs/plugin/timezone";
import { useGDCStore } from "../stores/gdc";
import { useLDCStore } from "../stores/ldc";
import type { GDC, LDC } from '@/gen/api_pb';
import { storeToRefs } from "pinia";
import duckApiClient from "@/api/client";

dayjs.extend(utc);
dayjs.extend(timezone);

export type RateMeasurementType = {
  Timestamp: number;
  Value: number;
};

export type MessageType = {
  host: string;
  timestamp: string;
  type: string;
  value: string;
  run: number;
};

export type MetricsType = {
  EventCounter: number;
  ByteCounter: number;
  CurrentDataRate: number;
  CurrentTrgRate: number;
  AvgDataRate: number;
  AvgTrgRate: number;
};

export type StateType = {
  state: string;
};

export type OutputFileType = {
  server: string;
  subrun: number;
};

export type SummaryType = {
  host: string;
  events: number;
  bytes: number;
  errors: number;
};

export const useMessagesStore = defineStore("messages", () => {
  const gdcStore = useGDCStore();
  const ldcStore = useLDCStore();
  
  gdcStore.getGDCs();
  ldcStore.getLDCs();
  const { gdcs, errorGDCs } = storeToRefs(gdcStore);
  const { ldcs, errorLDCs } = storeToRefs(ldcStore);

  const messages: Ref<Array<MessageType>> = ref([]);
  const messagesDebug: Ref<Array<MessageType>> = ref([]);

  const evts: Ref<Record<string, number>> = ref({});
  const bytes: Ref<Record<string, number>> = ref({});
  const evtRateAvg: Ref<Record<string, number>> = ref({});
  const evtRateCurrent: Ref<Record<string, number>> = ref({});
  const byteRateAvg: Ref<Record<string, number>> = ref({});
  const byteRateCurrent: Ref<Record<string, number>> = ref({});

  // One entry for all GDCs and one entry for all LDCs
  const evtsTotal: Ref<Record<string, number>> = ref({});
  const bytesTotal: Ref<Record<string, number>> = ref({});
  const evtRateAvgTotal: Ref<Record<string, number>> = ref({});
  const evtRateCurrentTotal: Ref<Record<string, number>> = ref({});
  const byteRateAvgTotal: Ref<Record<string, number>> = ref({});
  const byteRateCurrentTotal: Ref<Record<string, number>> = ref({});

  const states: Ref<Record<string, string>> = ref({});
  const status: Ref<string> = ref("");
  const controlEnabled: Ref<boolean> = ref(false);
  const startRunEnabled: Ref<boolean> = ref(false);
  const stopRunEnabled: Ref<boolean> = ref(false);
  const runNumber: Ref<number> = ref(0);
  const errorRunNumber: Ref<string> = ref("");
  const errorStates: Ref<string> = ref("");

  const fileNumbers: Ref<Record<string, number>> = ref({});
  const waitForSummaries: Ref<boolean> = ref(false);
  const nSummariesReceived: Ref<number> = ref(0);

  const numberOfMessages = 200;
  const errorCentrifuge: Ref<string> = ref("");
  const warningCentrifuge: Ref<string> = ref("");

  function resetTotalCounters() {
    evtsTotal.value["gdc"] = 0;
    bytesTotal.value["gdc"] = 0;
    byteRateAvgTotal.value["gdc"] = 0;
    byteRateCurrentTotal.value["gdc"] = 0;
    evtRateAvgTotal.value["gdc"] = 0;
    evtRateCurrentTotal.value["gdc"] = 0;

    evtsTotal.value["ldc"] = 0;
    bytesTotal.value["ldc"] = 0;
    byteRateAvgTotal.value["ldc"] = 0;
    byteRateCurrentTotal.value["ldc"] = 0;
    evtRateAvgTotal.value["ldc"] = 0;
    evtRateCurrentTotal.value["ldc"] = 0;
  }

  resetTotalCounters();

  function resetCounters(servers: Array<GDC | LDC> | { name: string }[]) {
    if (!Array.isArray(servers)) {
      return
    }
    servers.forEach((server) => {
      evts.value[server.name] = 0
      bytes.value[server.name] = 0
      byteRateAvg.value[server.name] = 0
      byteRateCurrent.value[server.name] = 0
      evtRateAvg.value[server.name] = 0
      evtRateCurrent.value[server.name] = 0
      fileNumbers.value[server.name] = 0
    })
    resetTotalCounters()
  }

  function restartRun() {
    resetCounters(gdcs.value as Array<GDC | LDC>)
    resetCounters(ldcs.value as Array<GDC | LDC>)
    nSummariesReceived.value = 0
    waitForSummaries.value = false
  }

  watch(gdcs, (values) => {
    resetCounters(gdcs.value as Array<GDC | LDC>)
  })

  watch(ldcs, (values) => {
    resetCounters(ldcs.value as Array<GDC | LDC>)
  })

  async function getToken() {
    const response = await duckApiClient.getToken({});
    return response.token;
  }

  let centrifuge: any = null;
  let sub: any = null;

  // Use WebSocket transport endpoint.
  let baseURL = import.meta.env.VITE_WS_SERVER;
  if (baseURL == undefined || baseURL == "") {
    const currentHost = window.location.hostname;
    const currentPort = window.location.port;
    const currentProtocol = window.location.protocol;
    baseURL = `ws://${currentHost}:${currentPort}/daq`;
    if (currentProtocol == "https:") {
      baseURL = `wss://${currentHost}:${currentPort}/daq`;
    }
  }
  console.log(baseURL);
  centrifuge = new Centrifuge(`${baseURL}/connection/websocket`, {
    getToken: getToken,
  });
  // Allocate Subscription to a channel.
  sub = centrifuge.newSubscription("duck");

  // Trigger subscribe process.
  sub.subscribe();

  // Trigger actual connection establishement.
  centrifuge.connect();

  if (centrifuge) {
    centrifuge.on("connected", function (ctx: any) {
      // now client connected to Centrifugo and authenticated.
      errorCentrifuge.value = "";
      warningCentrifuge.value = "";
      getStates();
    });

    centrifuge.on("connecting", function (ctx: any) {
      // do whatever you need in case of connecting to a server
      warningCentrifuge.value = "Connecting to server...";
    });

    centrifuge.on("disconnected", function (ctx: any) {
      // do whatever you need in case of disconnect from server
      errorCentrifuge.value = "Disconnected from server, please refresh the page";
      warningCentrifuge.value = "";
    });

    centrifuge.on("error", function (ctx: any) {
      errorCentrifuge.value = "Error in server connection, retrying...";
      warningCentrifuge.value = "";
      console.log("error on centrifuge connection");
      console.log(ctx);
    });
  }

  if (sub) {
    sub.on("publication", function (ctx: any) {
    //console.log(ctx.data);
    //const evtParsed = JSON.parse(ctx.data)
    const evtParsed = ctx.data;
    //if (evtParsed.type != "metric") {
    //  console.log(evtParsed);
    //}
    try {
      switch (evtParsed.type) {
        case "metric":
          const data: MetricsType = JSON.parse(evtParsed.value);

          // Update values
          evts.value[evtParsed.host] = data.EventCounter;
          bytes.value[evtParsed.host] = data.ByteCounter;
          byteRateCurrent.value[evtParsed.host] = data.CurrentDataRate;
          evtRateCurrent.value[evtParsed.host] = data.CurrentTrgRate;
          byteRateAvg.value[evtParsed.host] = data.AvgDataRate;
          evtRateAvg.value[evtParsed.host] = data.AvgTrgRate;

          updateTotalCounters();
          break;
        case "debug":
          messagesDebug.value.push(evtParsed);
          if (messagesDebug.value.length > numberOfMessages) {
            messagesDebug.value.splice(
              0,
              messagesDebug.value.length - numberOfMessages
            );
          }
          break;
        case "state":
          const stateData: StateType = JSON.parse(evtParsed.value);
          states.value[evtParsed.host] = stateData.state;

          const stateValues = Object.values(states.value);
          const firstState = stateValues[0];
          const allStatesEqual = stateValues.every(
            (state) => state === firstState
          );
          if (allStatesEqual && firstState) {
            status.value = firstState;
            controlEnabled.value = true;
            if (firstState == "INITIALIZED") {
              startRunEnabled.value = true;
              stopRunEnabled.value = false;
            }
            if (firstState == "RUNNING") {
              startRunEnabled.value = false;
              stopRunEnabled.value = true;
            }
          } else {
            status.value = "Changing state...";
            controlEnabled.value = false;
            startRunEnabled.value = false;
            stopRunEnabled.value = false;
          }
          break;
        case "output":
          const fileData: OutputFileType = JSON.parse(evtParsed.value);
          fileNumbers.value[evtParsed.host] = fileData.subrun;
          break;
        case "summary":
          const summaryData: SummaryType = JSON.parse(evtParsed.value);
          nSummariesReceived.value += 1;
          if (
            nSummariesReceived.value ==
            gdcStore.nActiveGDCs + ldcStore.nActiveLDCs
          ) {
            waitForSummaries.value = false;
          }
          const summaryStr = `received ${summaryData.events} and ${summaryData.bytes} bytes`;
          const evtToPrint: MessageType = {
            host: evtParsed.host,
            timestamp: dayjs().tz("Europe/Madrid").format("YYYY-MM-DD HH:mm:ss"),
            type: "summary",
            value: summaryStr,
            run: 0,
          };
          messages.value.push(evtToPrint);
          if (messages.value.length > numberOfMessages) {
            messages.value.splice(0, messages.value.length - numberOfMessages);
          }
          break;
        default:
          messages.value.push(evtParsed);
          if (messages.value.length > numberOfMessages) {
            messages.value.splice(0, messages.value.length - numberOfMessages);
          }
      }
    } catch (error) {
      console.error("Error processing message:", error);
      // Silently ignore malformed messages
    }
    });
  }

  function updateTotalCounters() {
    evtsTotal.value["gdc"] = 0;
    bytesTotal.value["gdc"] = 0;
    evtRateAvgTotal.value["gdc"] = 0;
    evtRateCurrentTotal.value["gdc"] = 0;
    byteRateAvgTotal.value["gdc"] = 0;
    byteRateCurrentTotal.value["gdc"] = 0;

    // Update total counters for GDC
    Object.keys(evts.value).forEach((key) => {
      if (key.startsWith("gdc")) {
        evtsTotal.value["gdc"] += evts.value[key];
        bytesTotal.value["gdc"] += bytes.value[key];
        evtRateAvgTotal.value["gdc"] += evtRateAvg.value[key];
        evtRateCurrentTotal.value["gdc"] += evtRateCurrent.value[key];
        byteRateAvgTotal.value["gdc"] += byteRateAvg.value[key];
        byteRateCurrentTotal.value["gdc"] += byteRateCurrent.value[key];
      }
    });

    evtsTotal.value["ldc"] = 0;
    bytesTotal.value["ldc"] = 0;
    evtRateAvgTotal.value["ldc"] = 0;
    evtRateCurrentTotal.value["ldc"] = 0;
    byteRateAvgTotal.value["ldc"] = 0;
    byteRateCurrentTotal.value["ldc"] = 0;

    // Update total counters for LDC
    Object.keys(evts.value).forEach((key) => {
      if (key.startsWith("ldc")) {
        evtsTotal.value["ldc"] += evts.value[key];
        bytesTotal.value["ldc"] += bytes.value[key];
        evtRateAvgTotal.value["ldc"] += evtRateAvg.value[key];
        evtRateCurrentTotal.value["ldc"] += evtRateCurrent.value[key];
        byteRateAvgTotal.value["ldc"] += byteRateAvg.value[key];
        byteRateCurrentTotal.value["ldc"] += byteRateCurrent.value[key];
      }
    });
  }

  async function getStates() {
    try {
      const response = await duckApiClient.getProcessStates({})
      console.log(response)
      errorStates.value = ""
    } catch (error: any) {
      console.log(error)
      errorStates.value = error.message
    }
  }

  async function getRunNumber() {
    try {
      const response = await duckApiClient.getRunNumber({})
      runNumber.value = Number(response.runNumber)
      errorRunNumber.value = ""
    } catch (error: any) {
      console.log(error)
      errorRunNumber.value = error.message
    }
  }

  getRunNumber();

  return {
    messages,
    messagesDebug,
    evts,
    bytes,
    evtRateAvg,
    evtRateCurrent,
    byteRateAvg,
    byteRateCurrent,

    evtsTotal,
    bytesTotal,
    evtRateAvgTotal,
    evtRateCurrentTotal,
    byteRateAvgTotal,
    byteRateCurrentTotal,

    states,
    fileNumbers,
    status,
    runNumber,
    controlEnabled,
    startRunEnabled,
    stopRunEnabled,
    errorRunNumber,
    errorStates,
    errorCentrifuge,
    warningCentrifuge,
    waitForSummaries,
    nSummariesReceived,
    getRunNumber,
    restartRun,
  };
});
