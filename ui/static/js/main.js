function formatResponse(data, isError = false) {
    const className = isError ? "error" : "success";
    let content;

    if (typeof data === "object") {
        content = `<pre>${JSON.stringify(data, null, 4)}</pre>`;
    } else {
        content = data;
    }

    return { className, content };
}

async function sendRequest(url, method, responseElementId, body = null) {
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

        console.log("Sending request:", { url, method, body: options.body });

        const response = await fetch(url, options);
        const data = await (response.headers
            .get("Content-Type")
            ?.includes("application/json")
            ? response.json()
            : response.text());

        const responseElement = document.getElementById(responseElementId);
        if (!responseElement) return;

        if (!response.ok) {
            const { className, content } = formatResponse(
                data || "An error occurred",
                true
            );
            responseElement.className = `command-response ${className}`;
            responseElement.innerHTML = content;
        } else {
            const { className, content } = formatResponse(data);
            responseElement.className = `command-response ${className}`;
            responseElement.innerHTML = content;
        }
    } catch (error) {
        console.log(error);
        const responseElement = document.getElementById(responseElementId);
        if (responseElement) {
            const { className, content } = formatResponse(error.message, true);
            responseElement.className = `command-response ${className}`;
            responseElement.innerHTML = content;
        }
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

    await sendRequest("/set", "POST", "setResponse", body);
});

// GET Command
document.getElementById("getForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    const key = document.getElementById("getKey").value;
    await sendRequest(
        `/get?key=${encodeURIComponent(key)}`,
        "GET",
        "getResponse"
    );
});

// DELETE Command
document.getElementById("deleteForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    const keys = document
        .getElementById("deleteKeys")
        .value.split(",")
        .map((k) => k.trim())
        .filter((k) => k);
    await sendRequest("/delete", "POST", "deleteResponse", { keys });
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
    await sendRequest("/push", "POST", "lpushResponse", {
        key,
        values,
        direction: "left",
    });
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
    await sendRequest("/push", "POST", "rpushResponse", {
        key,
        values,
        direction: "right",
    });
});

// LPOP Command
document.getElementById("lpopForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    const key = document.getElementById("lpopKey").value;
    await sendRequest("/pop", "POST", "lpopResponse", {
        key,
        direction: "left",
    });
});

// RPOP Command
document.getElementById("rpopForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    const key = document.getElementById("rpopKey").value;
    await sendRequest("/pop", "POST", "rpopResponse", {
        key,
        direction: "right",
    });
});

// LLEN Command
document.getElementById("llenForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    const key = document.getElementById("llenKey").value;
    await sendRequest(
        `/llen?key=${encodeURIComponent(key)}`,
        "GET",
        "llenResponse"
    );
});

// LRANGE Command
document.getElementById("lrangeForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    const key = document.getElementById("lrangeKey").value;
    const start = document.getElementById("lrangeStart").value;
    const end = document.getElementById("lrangeEnd").value;
    await sendRequest(
        `/lrange?key=${encodeURIComponent(key)}&start=${start}&end=${end}`,
        "GET",
        "lrangeResponse"
    );
});

// EXPIRE Command
document.getElementById("expireForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    const key = document.getElementById("expireKey").value;
    const expireSeconds = document.getElementById("expireSeconds").value;

    await sendRequest("/expires", "POST", "expireResponse", {
        key: key,
        expiration: parseInt(expireSeconds),
    });
});
