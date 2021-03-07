const fetch = require("node-fetch");

(async () => {
  const resp = await fetch("http://localhost:8080/configurations", {
    method: "POST",
    body: JSON.stringify(require("./test-configuration.json")),
  });

  if (!resp.ok) {
    throw new Error("Failed to apply test configuration", resp.body);
  }
  console.log("Hi");

  const array = Array(10000).fill(0);
  await Promise.all(
    array.map((item, index) =>
      fetch("http://localhost:8080/data-streams/cars/documents", {
        method: "POST",
        body: JSON.stringify({
          owner: "Hello-" + index,
          quantity: index,
          test: "AAA",
          type: "cars",
        }),
      })
    )
  );

  console.log("Hi");
})().catch((e) => {
  console.error(e);
});
