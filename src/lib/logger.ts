import Pino from "pino";

const logger = Pino({
  transport: {
    target: "pino-pretty",
  },
  level: "error",
});

export default logger;