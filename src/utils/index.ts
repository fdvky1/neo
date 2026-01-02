const parseMention = (text = "") => [...text.toString().matchAll(/@([0-9]{5,16}|0)/g)].map((v) => v[1] + "@s.whatsapp.net");


export { parseMention };