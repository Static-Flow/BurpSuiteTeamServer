function getRooms() {
    fetch('/rooms').then(function (response) {
        return response.json();
    }).then(function (rooms) {
        const allRoomsDiv = document.getElementById('rooms');
        while (allRoomsDiv.firstChild) {
            allRoomsDiv.removeChild(allRoomsDiv.firstChild);
        }
        for (let room in rooms) {
            console.log(rooms[room]);
            let roomDiv = document.createElement("div");
            let roomDivText = document.createTextNode(rooms[room]);
            roomDiv.appendChild(roomDivText);
            allRoomsDiv.appendChild(roomDiv);
        }
    });
}

function createRoom() {
    postData('/newroom', {})
        .then(data => console.log(JSON.stringify(data)))
        .catch(error => console.error(error));
}

function postData(url = '', data = {}) {
    return fetch(url, {
        method: 'POST',
        mode: 'cors',
        cache: 'no-cache',
        credentials: 'same-origin',
        headers: {
            'Content-Type': 'application/json',
        },
        redirect: 'follow',
        referrer: 'no-referrer',
        body: JSON.stringify(data),
    }).then(response => response.json());
}

getRooms();