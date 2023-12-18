const addResourcesToCache = async (resources) => {
    const cache = await caches.open("v1");
    await cache.addAll(resources);
  };
  
  self.addEventListener("install", (event) => {
    event.waitUntil(
      addResourcesToCache([
        "/index.html",
        "/css/mdui.min.css",
        "/js/mdui.min.js",
        "/js/changebrightness.js",
        "/admin/index.html",
        "/icons/material-icons/MaterialIcons-Regular.ijmap",
        "/icons/material-icons/MaterialIcons-Regular.woff",
        "/icons/material-icons/MaterialIcons-Regular.woff2"
      ]),
    );
  });