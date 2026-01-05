'use client';

import React, { useState } from 'react';
import { Search, UserPlus, Loader2 } from 'lucide-react';
import { Modal } from '@/components/ui/Modal';
import { Input } from '@/components/ui/Input';
import { Button } from '@/components/ui/Button';
import { Avatar } from '@/components/ui/Avatar';
import { contactsApi, User } from '@/lib/api';
import { useChatStore } from '@/stores/chatStore';

interface AddContactModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export function AddContactModal({ isOpen, onClose }: AddContactModalProps) {
  const { fetchContacts } = useChatStore();
  const [uniqueId, setUniqueId] = useState('');
  const [searchResult, setSearchResult] = useState<User | null>(null);
  const [isSearching, setIsSearching] = useState(false);
  const [isAdding, setIsAdding] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  const handleSearch = async () => {
    if (!uniqueId.trim()) return;
    
    setError('');
    setSuccess('');
    setSearchResult(null);
    setIsSearching(true);

    try {
      const response = await contactsApi.search(uniqueId.trim());
      setSearchResult(response.user);
    } catch {
      setError('User not found');
    } finally {
      setIsSearching(false);
    }
  };

  const handleAddContact = async () => {
    if (!searchResult) return;

    setIsAdding(true);
    setError('');

    try {
      await contactsApi.add(searchResult.unique_id);
      setSuccess('Contact added successfully!');
      fetchContacts();
      setTimeout(() => {
        handleClose();
      }, 1500);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Failed to add contact';
      setError(message);
    } finally {
      setIsAdding(false);
    }
  };

  const handleClose = () => {
    setUniqueId('');
    setSearchResult(null);
    setError('');
    setSuccess('');
    onClose();
  };

  return (
    <Modal isOpen={isOpen} onClose={handleClose} title="Add Contact">
      <div className="space-y-4">
        <p className="text-sm text-[var(--text-tertiary)]">
          Enter your friend&apos;s unique ID to add them to your contacts.
        </p>

        <div className="flex gap-2">
          <Input
            type="text"
            placeholder="#GOPRO-882"
            value={uniqueId}
            onChange={(e) => setUniqueId(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
            leftIcon={<Search className="w-4 h-4" />}
          />
          <Button
            onClick={handleSearch}
            isLoading={isSearching}
            disabled={!uniqueId.trim()}
          >
            Search
          </Button>
        </div>

        {error && (
          <p className="text-sm text-[var(--accent-red)]">{error}</p>
        )}

        {success && (
          <p className="text-sm text-[var(--accent-green)]">{success}</p>
        )}

        {searchResult && !success && (
          <div className="p-4 bg-[var(--bg-secondary)] rounded-xl">
            <div className="flex items-center gap-3">
              <Avatar src={searchResult.avatar} name={searchResult.name} size="lg" />
              <div className="flex-1">
                <h3 className="font-medium text-[var(--text-primary)]">{searchResult.name}</h3>
                <p className="text-sm text-[var(--text-tertiary)]">{searchResult.unique_id}</p>
              </div>
              <Button
                onClick={handleAddContact}
                isLoading={isAdding}
                leftIcon={<UserPlus className="w-4 h-4" />}
              >
                Add
              </Button>
            </div>
          </div>
        )}
      </div>
    </Modal>
  );
}
