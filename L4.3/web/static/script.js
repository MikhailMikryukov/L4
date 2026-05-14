const API_BASE = 'http://localhost:8080';

async function fetchEvents() {
    try {
        const response = await fetch(`${API_BASE}/events`);
        if (!response.ok) throw new Error('Failed to fetch events');
        const events = await response.json();
        displayEvents(events);
    } catch (error) {
        console.error('Error fetching events:', error);
        document.getElementById('eventsList').innerHTML = '<div class="error">Failed to load events. Make sure the server is running.</div>';
    }
}

async function fetchArchivedEvents() {
    try {
        const response = await fetch(`${API_BASE}/archived_events`);
        if (!response.ok) throw new Error('Failed to fetch events');
        const events = await response.json();
        displayArchivedEvents(events);
    } catch (error) {
        console.error('Error fetching events:', error);
        document.getElementById('archivedEventsList').innerHTML = '<div class="error">Failed to load events. Make sure the server is running.</div>';
    }
}

function displayEvents(events) {
    const container = document.getElementById('eventsList');

    if (!events || events.length === 0) {
        container.innerHTML = '<div class="empty-state"> No events found. Create your first event!</div>';
        return;
    }

    container.innerHTML = events.map(event => `
        <div class="event-item">
            <div class="event-header">
                <div>
                    <span class="event-name">${escapeHtml(event.name)}</span>
                    <span class="notification-badge ${event.ShouldNotify ? 'notification-yes' : 'notification-no'}">
                        ${event.should_notify ? 'Notify' : 'No notify'}
                    </span>
                </div>
                <span class="event-id">ID: ${event.id}</span>
            </div>
            <div class="event-details">
                <div>Date: ${formatDate(event.date)}</div>
                <div>User: ${escapeHtml(event.user_id)}</div>
                ${event.should_notify ? `<div>Notify at: ${formatDate(event.notify_at)}</div>` : ''}
                <div>Created: ${formatDate(event.created_at)}</div>
                <div>Updated: ${formatDate(event.updated_at)}</div>
            </div>
            ${event.description ? `<div class="event-description">${escapeHtml(event.Description)}</div>` : ''}
            <div class="event-actions">
                <button onclick='fillUpdateForm(${JSON.stringify(event)})' class="update-btn" style="background: #48bb78;">Update</button>
                <button onclick="deleteEventById(${event.id})" class="delete-btn">Delete</button>
            </div>
        </div>
    `).join('');
}

function displayArchivedEvents(events) {
    const container = document.getElementById('archivedEventsList');

    if (!events || events.length === 0) {
        container.innerHTML = '<div class="empty-state"> No archived events found.</div>';
        return;
    }

    container.innerHTML = events.map(archivedEvent => `
        <div class="event-item">
            <div class="event-header">
                <div>
                    <span class="event-name">${escapeHtml(archivedEvent.name)}</span>
                </div>
                <span class="event-id">ID: ${archivedEvent.id}</span>
            </div>
            <div class="event-details">
                <div>Event id: ${archivedEvent.event_id}</div>
                <div>Date: ${formatDate(archivedEvent.date)}</div>
                <div>User: ${escapeHtml(archivedEvent.user_id)}</div>
                <div>Created: ${formatDate(archivedEvent.created_at)}</div>
                <div>Archived: ${formatDate(archivedEvent.archived_at)}</div>
            </div>
        </div>
    `).join('');
}

function fillUpdateForm(event) {
    // Заполняем ID
    document.getElementById('updateId').value = event.id;

    // Заполняем остальные поля текущими значениями
    document.getElementById('updateName').value = event.name || '';
    document.getElementById('updateUserID').value = event.user_id || '';
    document.getElementById('updateDescription').value = event.description || '';

    // Заполняем даты (форматируем для input datetime-local)
    if (event.date) {
        document.getElementById('updateDate').value = formatDateTimeLocal(event.date);
    } else {
        document.getElementById('updateDate').value = '';
    }

    if (event.notify_at) {
        document.getElementById('updateNotifyAt').value = formatDateTimeLocal(event.notify_at);
    } else {
        document.getElementById('updateNotifyAt').value = '';
    }

    // Заполняем чекбокс уведомления
    document.getElementById('updateShouldNotify').checked = event.should_notify || false;

    // Прокручиваем к форме
    document.getElementById('updateId').scrollIntoView({ behavior: 'smooth', block: 'center' });
}

// Вспомогательная функция для форматирования даты в формат datetime-local
function formatDateTimeLocal(dateString) {
    if (!dateString) return '';

    const date = new Date(dateString);
    if (isNaN(date.getTime())) return '';

    // Формат: YYYY-MM-DDThh:mm:ss
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    const hours = String(date.getHours()).padStart(2, '0');
    const minutes = String(date.getMinutes()).padStart(2, '0');
    const seconds = String(date.getSeconds()).padStart(2, '0');

    return `${year}-${month}-${day}T${hours}:${minutes}:${seconds}`;
}

async function deleteEventById(id) {
    if (!confirm(`Are you sure you want to delete event #${id}?`)) return;

    try {
        const response = await fetch(`${API_BASE}/delete_event`, {
            method: 'DELETE',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({id: id})
        });

        const result = await response.json();
        if (response.ok) {
            showMessage('Event deleted successfully!', 'success');
            fetchEvents();
        } else {
            showMessage(result.error || 'Failed to delete event', 'error');
        }
    } catch (error) {
        showMessage('Error deleting event: ' + error.message, 'error');
    }
}

function formatDate(dateString) {
    if (!dateString) return 'N/A';
    const date = new Date(dateString);
    return date.toLocaleString();
}

function escapeHtml(str) {
    if (!str) return '';
    return str
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}

function showMessage(message, type) {
    const messageDiv = document.createElement('div');
    messageDiv.className = type === 'error' ? 'error' : 'success';
    messageDiv.textContent = message;

    const container = document.querySelector('.dashboard');
    container.insertBefore(messageDiv, container.firstChild);

    setTimeout(() => messageDiv.remove(), 3000);
}

// Create Event
document.getElementById('createEventForm').addEventListener('submit', async (e) => {
    e.preventDefault();
   let  notifyAt = document.getElementById('createNotifyAt').value

    const eventData = {
        name: document.getElementById('createName').value,
        date: document.getElementById('createDate').value + ":00Z",
        user_id: document.getElementById('createUserID').value,
        should_notify: document.getElementById('createShouldNotify').checked,
        notify_at: notifyAt === "" ? null : notifyAt + ":00Z",
        description: document.getElementById('createDescription').value
    };

    try {
        const response = await fetch(`${API_BASE}/create_event`, {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify(eventData)
        });

        const result = await response.json();
        if (response.ok) {
            showMessage('Event created successfully!', 'success');
            e.target.reset();
            fetchEvents();
        } else {
            showMessage(result.error || 'Failed to create event', 'error');
        }
    } catch (error) {
        showMessage('Error creating event: ' + error.message, 'error');
    }
});


// Update Event
document.getElementById('updateEventForm').addEventListener('submit', async (e) => {
    e.preventDefault();

    let  notifyAt = document.getElementById('createNotifyAt').value
    let  dateUpd = document.getElementById('updateDate').value

    const updateData = {
        id: parseInt(document.getElementById('updateId').value),
        name: document.getElementById('updateName').value || undefined,
        date: dateUpd === "" ? null : dateUpd + ":00Z",
        user_id: document.getElementById('updateUserID').value || undefined,
        should_notify: document.getElementById('updateShouldNotify').checked,
        notify_at: notifyAt === "" ? null : notifyAt + ":00Z",
        description: document.getElementById('updateDescription').value || undefined
    };

    try {
        const response = await fetch(`${API_BASE}/update_event`, {
            method: 'PUT',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify(updateData)
        });

        const result = await response.json();
        if (response.ok) {
            showMessage('Event updated successfully!', 'success');
            e.target.reset();
            fetchEvents();
        } else {
            showMessage(result.error || 'Failed to update event', 'error');
        }
    } catch (error) {
        showMessage('Error updating event: ' + error.message, 'error');
    }
});


// Delete Event
document.getElementById('deleteEventForm').addEventListener('submit', async (e) => {
    e.preventDefault();

    const id = parseInt(document.getElementById('deleteId').value);
    if (!confirm(`Are you sure you want to delete event #${id}?`)) return;

    try {
        const response = await fetch(`${API_BASE}/delete_event`, {
            method: 'DELETE',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({id: id})
        });

        const result = await response.json();
        if (response.ok) {
            showMessage('Event deleted successfully!', 'success');
            e.target.reset();
            fetchEvents();
        } else {
            showMessage(result.error || 'Failed to delete event', 'error');
        }
    } catch (error) {
        showMessage('Error deleting event: ' + error.message, 'error');
    }
});


// Initial load
fetchEvents();
fetchArchivedEvents()

// Refresh events every 5 seconds
setInterval(fetchEvents, 5000);
setInterval(fetchArchivedEvents, 5000);