const application = Stimulus.Application.start();

class LogsController extends Stimulus.Controller {
  connect() {
    console.log("Connecting to logs event source.");
    this.eventSource = new EventSource("/logs/events");
    this.eventSource.onopen = this.onOpen;
    this.eventSource.onmessage = this.onMessage.bind(this);
    this.eventSource.onerror = this.onError;
  }

  onOpen(event) {
    console.log("Connected to logs event source.");
  }

  onMessage(message) {
    console.log("onMessage", message);
    var newElement = document.createElement("p");
    newElement.textContent = message.data;
    this.element.appendChild(newElement);
  }

  onError(error) {
    console.error("Logs event source error.", error);
  }

  disconnect() {
    console.log("Closing logs event source.");
    this.eventSource.close();
  }
}
application.register("logs", LogsController);

class BytesController extends Stimulus.Controller {
  connect() {
    if (!this.data.get("done")) {
      this.element.textContent = byteSize(this.element.textContent, {
        units: this.data.get("format")
      });
    }
    this.data.set("done", true);
  }
}
application.register("bytes", BytesController);

class ByteSize {
  constructor(bytes, options) {
    options = options || {};
    options.units = options.units || "metric";
    options.precision =
      typeof options.precision === "undefined" ? 1 : options.precision;

    const table = [
      {
        expFrom: 0,
        expTo: 1,
        metric: "B",
        iec: "B",
        metric_octet: "o",
        iec_octet: "o"
      },
      {
        expFrom: 1,
        expTo: 2,
        metric: "kB",
        iec: "KiB",
        metric_octet: "ko",
        iec_octet: "Kio"
      },
      {
        expFrom: 2,
        expTo: 3,
        metric: "MB",
        iec: "MiB",
        metric_octet: "Mo",
        iec_octet: "Mio"
      },
      {
        expFrom: 3,
        expTo: 4,
        metric: "GB",
        iec: "GiB",
        metric_octet: "Go",
        iec_octet: "Gio"
      },
      {
        expFrom: 4,
        expTo: 5,
        metric: "TB",
        iec: "TiB",
        metric_octet: "To",
        iec_octet: "Tio"
      },
      {
        expFrom: 5,
        expTo: 6,
        metric: "PB",
        iec: "PiB",
        metric_octet: "Po",
        iec_octet: "Pio"
      },
      {
        expFrom: 6,
        expTo: 7,
        metric: "EB",
        iec: "EiB",
        metric_octet: "Eo",
        iec_octet: "Eio"
      },
      {
        expFrom: 7,
        expTo: 8,
        metric: "ZB",
        iec: "ZiB",
        metric_octet: "Zo",
        iec_octet: "Zio"
      },
      {
        expFrom: 8,
        expTo: 9,
        metric: "YB",
        iec: "YiB",
        metric_octet: "Yo",
        iec_octet: "Yio"
      }
    ];

    const base =
      options.units === "metric" || options.units === "metric_octet"
        ? 1000
        : 1024;
    const prefix = bytes < 0 ? "-" : "";
    bytes = Math.abs(bytes);

    for (let i = 0; i < table.length; i++) {
      const lower = Math.pow(base, table[i].expFrom);
      const upper = Math.pow(base, table[i].expTo);
      if (bytes >= lower && bytes < upper) {
        const units = table[i][options.units];
        if (i === 0) {
          this.value = prefix + bytes;
          this.unit = units;
          return;
        } else {
          this.value = prefix + (bytes / lower).toFixed(options.precision);
          this.unit = units;
          return;
        }
      }
    }
    this.value = prefix + bytes;
    this.unit = "";
  }

  toString() {
    return `${this.value} ${this.unit}`.trim();
  }
}

function byteSize(bytes, options) {
  return new ByteSize(bytes, options);
}

class EventsController extends Stimulus.Controller {
  connect() {
    console.log("Events controller connected.");
    this.eventSource = new EventSource("/events");
    this.eventSource.onopen = this.onOpen;
    this.eventSource.onmessage = this.onMessage.bind(this);
    this.eventSource.onerror = this.onError;
  }

  onOpen(event) {
    console.log("Connected events source.");
  }

  onMessage(message) {
    console.log("onMessage", message);
  }

  onError(error) {
    console.error("Events source error.", error);
  }

  disconnect() {
    console.log("Closing events source.");
    this.eventSource.close();
  }
}
application.register("events", EventsController);
