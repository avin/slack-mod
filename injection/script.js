function waitForElement(selector, callback) {
    const intervalId = setInterval(() => {
        const element = document.querySelector(selector);
        if (element) {
            clearInterval(intervalId);
            callback(element);
        }
    }, 100);
}

waitForElement('.p-ia4_channel_list', (element) => {
    document.querySelector(".p-ia4_channel_list").classList.add("sk-client-theme--dark")

    let observer = new MutationObserver(mutations => {
        mutations.forEach(mutation => {
            // Проверка каждого добавленного узла
            for (let node of mutation.addedNodes) {
                node.querySelector(".p-ia4_channel_list:not(.sk-client-theme--dark)").classList.add("sk-client-theme--dark")
            }

        });
    });

    // Настройка observer и начало отслеживания изменений в DOM
    observer.observe(document.body, {
        childList: true, // отслеживание непосредственных дочерних элементов
        subtree: true,   // отслеживание всех потомков элемента
    });
});


