const fetch = require("node-fetch");
const faker = require("faker");

const chunk = (arr, size) =>
  Array.from({ length: Math.ceil(arr.length / size) }, (v, i) => arr.slice(i * size, i * size + size));

(async () => {
  const resp = await fetch("http://localhost:8080/api/configurations", {
    method: "POST",
    body: JSON.stringify(require("./test-configuration.json")),
  });

  if (!resp.ok) {
    throw new Error("Failed to apply test configuration", resp.body);
  }
  console.log("Hi");
  const array = Array(100000).fill(0);
  const data = chunk(array, 10000);

  for (let i = 0; i < 5; i++) {
    // await Promise.all(
    data.forEach(async (item, index) => {
      const result = await fetch("http://localhost:8080/api/data-streams/cars/documents", {
        method: "POST",
        body: JSON.stringify(
          item.map((item, i2) => ({
            owner: faker.name.findName(),
            quantity: index * i2 + i2,
            test: "AAA",
            type: faker.animal.type(),
          }))
        ),
      });

      console.log("Finished call");
    });
  }

  console.log("Hi");
})().catch((e) => {
  console.error(e);
});
