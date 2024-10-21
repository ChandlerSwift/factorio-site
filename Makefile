factorio-site:
	go build

.PHONY: deploy
deploy: factorio-site
	ssh factorio@zirconium systemctl --user stop factorio-site || true
	scp factorio-site factorio@zirconium:
	scp factorio-site.service factorio@zirconium:.config/systemd/user/
	ssh factorio@zirconium sh -c "systemctl --user daemon-reload >/dev/null && systemctl --user start factorio-site"

deploy-config: config.json
# TODO
