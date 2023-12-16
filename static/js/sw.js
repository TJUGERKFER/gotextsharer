const addResourcesToCache = async (resources) => {
    const cache = await caches.open("v1");
    await cache.addAll(resources);
  };
  
  self.addEventListener("install", (event) => {
    event.waitUntil(
      addResourcesToCache([
        "/",
        "/index.html",
        "/css/mdui.min.css",
        "/js/mdui.min.js",
        "/admin/index.html",
      ]),
    );
  });