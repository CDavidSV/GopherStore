const responseDiv = document.getElementById("response");

function displayResponse(data, isError = false) {
    responseDiv.className = isError ? "error" : "success";
    if (typeof data === "object") {
        responseDiv.innerHTML = `<pre>${JSON.stringify(data, null, 4)}</pre>`;
    } else {
        responseDiv.textContent = data;
    }
}

async function sendRequest(url, method, body = null) {
    try {
        const options = {
            method: method,
            headers: {
                "Content-Type": "application/json",
            },
        };

        if (body) {
            options.body = JSON.stringify(body);
        }

        const response = await fetch(url, options);
        const data = await (response.headers.get("Content-Type")?.includes("application/json") ? response.json() : response.text());

        if (!response.ok) {
            displayResponse(data || "An error occurred", true);
        } else {
            displayResponse(data);
        }
    } catch (error) {
        console.log(error);
        displayResponse(error.message, true);
    }
}

// --------- Handle Commands ---------

// SET Command
document.getElementById("setForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    const key = document.getElementById("setKey").value;
    const value = document.getElementById("setValue").value;
    const expiration = document.getElementById("setExpiration").value;
    const condition = document.getElementById("setCondition").value;

    const body = { key, value };
    if (expiration) body.expiration = parseInt(expiration);
    if (condition) body.condition = condition;

    await sendRequest("/set", "POST", body);
});

// GET Command
document.getElementById("getForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    const key = document.getElementById("getKey").value;
    await sendRequest(`/get?key=${encodeURIComponent(key)}`, "GET");
});

// DELETE Command
document.getElementById("deleteForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    const keys = document
        .getElementById("deleteKeys")
        .value.split(",")
        .map((k) => k.trim())
        .filter((k) => k);
    await sendRequest("/delete", "POST", { keys });
});

// LPUSH Command
document.getElementById("lpushForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    const key = document.getElementById("lpushKey").value;
    const values = document
        .getElementById("lpushValues")
        .value.split(",")
        .map((v) => v.trim())
        .filter((v) => v);
    await sendRequest("/push", "POST", { key, values, direction: "left" });
});

// RPUSH Command
document.getElementById("rpushForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    const key = document.getElementById("rpushKey").value;
    const values = document
        .getElementById("rpushValues")
        .value.split(",")
        .map((v) => v.trim())
        .filter((v) => v);
    await sendRequest("/push", "POST", { key, values, direction: "right" });
});

// LPOP Command
document.getElementById("lpopForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    const key = document.getElementById("lpopKey").value;
    await sendRequest("/pop", "POST", { key, direction: "left" });
});

// RPOP Command
document.getElementById("rpopForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    const key = document.getElementById("rpopKey").value;
    await sendRequest("/pop", "POST", { key, direction: "right" });
});

// LLEN Command
document.getElementById("llenForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    const key = document.getElementById("llenKey").value;
    await sendRequest(`/llen?key=${encodeURIComponent(key)}`, "GET");
});

// LRANGE Command
document.getElementById("lrangeForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    const key = document.getElementById("lrangeKey").value;
    const start = document.getElementById("lrangeStart").value;
    const end = document.getElementById("lrangeEnd").value;
    await sendRequest(
        `/lrange?key=${encodeURIComponent(key)}&start=${start}&end=${end}`,
        "GET"
    );
});

// EXPIRE Command
document.getElementById("expireForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    const key = document.getElementById("expireKey").value;
    const expireSeconds = document.getElementById("expireSeconds").value;
    await sendRequest("/expires", "POST", {
        key,
        expiration: parseInt(expireSeconds)
    });
});
