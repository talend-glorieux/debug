(() => {
  const application = Stimulus.Application.start();

  application.register("logs", class extends Stimulus.Controller {
    connect() {
      console.log("Connecting to logs event source.");
      this.eventSource = new EventSource("/logs/events");
      this.eventSource.onopen = this.onOpen;
      this.eventSource.onmessage = this.onMessage.bind(this);
      this.eventSource.onerror = this.onError;
    }

    onOpen(event) {
      console.log("Connection to logs event source opened.");
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
  });
})()
