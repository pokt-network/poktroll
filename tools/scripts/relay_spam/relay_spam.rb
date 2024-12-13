#!/usr/bin/env ruby

require 'yaml'
require 'optparse'
require 'json'
require 'fileutils'
require 'pty'
require 'timeout'

class RelaySpammer
  attr_reader :config, :options

  def initialize(options)
    @options = options
    @config = load_config(options.config_file)
  end

  def run
    case options.command
    when 'populate_config_accounts'
      populate_config_accounts
    when 'import_accounts'
      import_accounts
    when 'fund_accounts'
      fund_accounts
    when 'stake_applications'
      ensure_stake_applications
    when 'run'
      run_relay_spam
    else
      puts "Unknown command: #{options.command}"
      exit 1
    end
  end

  private

  def load_config(config_file)
    raw_config = YAML.load_file(config_file)
    
    # Filter out template keys that start with '.'
    raw_config.reject! { |k, _| k.start_with?('.') }
    
    # Expand the tx_flags template into individual commands
    if raw_config['tx_flags_template']
      template = raw_config['tx_flags_template']
      raw_config['tx_flags'] = template.map { |k,v| "--#{k}=#{v}" }.join(' ')
    end

    raw_config
  rescue => e
    puts "Error loading config file: #{e.message}"
    exit 1
  end

  def populate_config_accounts
    puts "Populating config with #{options.num_accounts} accounts..."
    
    temp_home = "/tmp/poktroll_temp_#{Time.now.to_i}"
    FileUtils.mkdir_p(temp_home)
    
    begin
      config_path = options.config_file
      existing_config = File.exist?(config_path) ? YAML.load_file(config_path) : @config
      
      existing_config['applications'] ||= []
      
      # Find the highest existing index
      existing_indices = existing_config['applications']
        .map { |app| app['name'].match(/relay_spam_app_(\d+)/)&.[](1).to_i }
        .compact
      start_index = (existing_indices.max || -1) + 1
      
      options.num_accounts.times do |i|
        key_name = "relay_spam_app_#{start_index + i}"
        create_key_cmd = [
          "poktrolld keys add",
          key_name,
          "--home=#{temp_home}",
          "--keyring-backend=#{config['keyring_backend']}",
          "--output json"
        ].join(' ')
        
        puts "Executing command: #{create_key_cmd}"
        
        output = `#{create_key_cmd} 2>&1`
        puts "Command output: #{output}"
        
        key_info = JSON.parse(output.lines.first)
        mnemonic = output.match(/mnemonic":"([^"]+)"/)[1]
        
        new_app = {
          'name' => key_name,
          'address' => key_info['address'],
          'mnemonic' => mnemonic
        }
        
        existing_config['applications'] << new_app
        puts "[#{i + 1}/#{options.num_accounts}] Added #{key_name}: #{key_info['address']}"
        
        File.write(config_path, existing_config.to_yaml)
      end
      
      puts "\nSuccessfully added #{options.num_accounts} accounts to config file"
    ensure
      FileUtils.rm_rf(temp_home)
    end
  end

  def import_accounts
    puts "Importing accounts from config..."
    
    config['applications'].each do |app|
      puts "\nProcessing #{app['name']}..."
      
      # First, try to delete the existing key
      delete_cmd = "poktrolld keys delete #{app['name']} --keyring-backend=#{config['keyring_backend']} -y"
      system(delete_cmd)
      
      # Now import the key
      import_cmd = "poktrolld keys add --recover #{app['name']} --keyring-backend=#{config['keyring_backend']}"
      
      begin
        PTY.spawn(import_cmd) do |stdout, stdin, pid|
          Timeout.timeout(10) do
            while true
              output = stdout.readline.strip
              puts output
              
              if output.include?("Enter your bip39 mnemonic")
                stdin.puts(app['mnemonic'])
                break
              end
            end
            
            # Read remaining output
            begin
              while (line = stdout.readline)
                puts line
              end
            rescue EOFError
              # Expected when process ends
            end
          end
        end
      rescue Timeout::Error
        puts "Timeout while processing #{app['name']}, moving to next..."
      rescue PTY::ChildExited => e
        puts "Process exited with status: #{e.status}"
      rescue Errno::EIO
        # This is expected when the process exits
      end
      
      puts "Processed #{app['name']}"
    end
    
    puts "\nFinished importing accounts"
  end

  def fund_accounts
    puts "Funding accounts..."
    
    # Get all addresses from applications
    addresses = config['applications'].map { |app| app['address'] }
    funding_amount = config['application_defaults']['funding_amount']
    
    # Build multi-send command
    fund_cmd = [
      "poktrolld tx bank multi-send",
      config['funder_address'],  # from address
      addresses.join(' '),       # all recipient addresses
      "#{funding_amount}upokt",  # amount each
      config['tx_flags'],        # common transaction flags
      "-y"                       # auto-confirm
    ].join(' ')
    
    puts "Executing funding command: #{fund_cmd}"
    success = system(fund_cmd)
    
    if success
      puts "\nSuccessfully funded #{addresses.length} accounts with #{funding_amount}upokt each"
    else
      puts "\nError funding accounts"
      exit 1
    end
  end

  def ensure_stake_applications
    total_apps = config['applications'].length
    puts "Ensuring applications are properly staked..."
    puts "Processing #{total_apps} applications..."

    config['applications'].each_with_index do |app, idx|
      print "\r[#{idx + 1}/#{total_apps}] Checking stake for #{app['name']} (#{app['address']})..."
      
      query_cmd = "poktrolld q application show-application #{app['address']} -o json"
      app_state = JSON.parse(`#{query_cmd} 2>/dev/null`) rescue nil
      
      if app_state.nil? || app_state['application'].nil?
        puts "\n  - Staking application..."
        stake_application(app)
      else
        current_stake = app_state['application']['stake']['amount'].to_s.gsub('upokt', '').to_i rescue 0
        expected_stake = config['application_defaults']['stake_amount'].to_s.gsub('upokt', '').to_i
        
        if current_stake < expected_stake
          puts "\n  - Updating stake amount (current: #{current_stake}, expected: #{expected_stake})"
          stake_application(app)
        end
      end
    end
    puts "\nStaking verification complete."

    # Then handle delegations gateway by gateway
    gateways = config['application_defaults']['gateways']
    puts "\nProcessing delegations for #{gateways.length} gateways..."

    gateways.each_with_index do |gateway_addr, gateway_idx|
      puts "\nGateway [#{gateway_idx + 1}/#{gateways.length}]: #{gateway_addr}"
      failed_apps = []

      config['applications'].each_with_index do |app, app_idx|
        print "\r  [#{app_idx + 1}/#{total_apps}] Checking #{app['name']}..."
        
        query_cmd = "poktrolld q application show-application #{app['address']} -o json"
        app_state = JSON.parse(`#{query_cmd} 2>/dev/null`) rescue nil
        current_gateways = app_state&.dig('application', 'delegatee_gateway_addresses') || []
        
        if !current_gateways.include?(gateway_addr)
          print " delegating..."
          if !delegate_to_gateway(app, gateway_addr)
            failed_apps << app
            print " failed"
          else
            print " ✓"
          end
        else
          print " already delegated ✓"
        end
      end

      # Retry failed delegations
      if failed_apps.any?
        puts "\n  Retrying #{failed_apps.length} failed delegations..."
        failed_apps.each do |app|
          print "\r  Retrying #{app['name']}..."
          if delegate_to_gateway(app, gateway_addr)
            print " ✓"
          else
            print " failed"
          end
        end
      end
    end
    
    puts "\nCompleted staking and delegation setup"
  end

  private

  def stake_application(app)
    stake_config = {
      'stake_amount' => config['application_defaults']['stake_amount'],
      'service_ids' => [config['application_defaults']['service_id']]
    }
    
    temp_config_path = "/tmp/stake_config_#{app['address']}.yaml"
    File.write(temp_config_path, stake_config.to_yaml)

    3.times do |attempt|
      stake_cmd = [
        "poktrolld tx application stake-application",
        "--config=#{temp_config_path}",
        "--from=#{app['address']}",
        config['tx_flags'],
        "-y"
      ].join(' ')

      puts "Attempt #{attempt + 1}: Staking application..."
      break if system(stake_cmd)
      sleep 5 # Wait before retry
    end

    FileUtils.rm_f(temp_config_path)
  end

  def delegate_to_gateway(app, gateway_addr)
    3.times do |attempt|
      delegate_cmd = [
        "poktrolld tx application delegate-to-gateway",
        gateway_addr,
        "--from=#{app['address']}",
        config['tx_flags'],
        "-y"
      ].join(' ')

      return true if system(delegate_cmd, out: File::NULL, err: File::NULL)
      sleep 5 unless attempt == 2
    end
    false
  end

  def run_relay_spam
    puts "Running relay spam..."
    
    # Create a Ractor for each application/gateway combination
    ractors = config['applications'].flat_map do |app|
      config['application_defaults']['gateways'].map do |gateway_addr|
        gateway_url = config['gateway_urls'][gateway_addr]
        
        Ractor.new(app, gateway_url, options.num_requests, options.concurrency) do |app, url, num_requests, concurrency|
          hey_cmd = [
            "hey",
            "-n #{num_requests}",
            "-c #{concurrency}",
            "-H 'Content-Type: application/json'",
            "-H 'X-App-Address: #{app['address']}'",
            "-m POST",
            "-d '{\"jsonrpc\":\"2.0\",\"method\":\"eth_blockNumber\",\"params\":[],\"id\":1}'",
            url
          ].join(' ')
          
          start_time = Time.now
          result = system(hey_cmd)
          end_time = Time.now
          
          # Return results to main thread
          {
            app_name: app['name'],
            gateway_url: url,
            success: result,
            duration: end_time - start_time
          }
        end
      end
    end
    
    # Create a progress monitor
    total_jobs = ractors.size
    completed_jobs = 0
    
    # Process results as they come in
    results = ractors.map do |r|
      result = r.take
      completed_jobs += 1
      
      # Print progress
      puts "[#{completed_jobs}/#{total_jobs}] #{result[:app_name]} -> #{result[:gateway_url]}: " +
           "#{result[:success] ? 'Success' : 'Failed'} (#{result[:duration].round(2)}s)"
      
      result
    end
    
    # Print summary
    successful = results.count { |r| r[:success] }
    puts "\nRelay spam completed:"
    puts "Total jobs: #{total_jobs}"
    puts "Successful: #{successful}"
    puts "Failed: #{total_jobs - successful}"
  end

  # Optional: Add a signal handler to gracefully shut down Ractors
  Signal.trap("INT") do
    puts "\nShutting down relay spam..."
    exit
  end
end

# Parse command line options
class Options
  attr_accessor :command, :config_file, :num_requests, :concurrency, :num_accounts

  def initialize
    @config_file = 'config.yml'
    @num_requests = 1000
    @concurrency = 50
    @num_accounts = 1000
    parse
  end

  private

  def parse
    OptionParser.new do |opts|
      opts.banner = "Usage: relay_spam.rb [options] COMMAND"

      opts.on("-c", "--config FILE", "Config file (default: config.yml)") do |f|
        @config_file = f
      end

      opts.on("-n", "--num-requests NUM", Integer, "Number of requests (default: 1000)") do |n|
        @num_requests = n
      end

      opts.on("-p", "--concurrency NUM", Integer, "Concurrent requests (default: 50)") do |c|
        @concurrency = c
      end

      opts.on("-a", "--num-accounts NUM", Integer, "Number of accounts to create (default: 1000)") do |a|
        @num_accounts = a
      end
    end.parse!

    @command = ARGV.shift
    unless ['populate_config_accounts', 'import_accounts', 'fund_accounts', 'stake_applications', 'run'].include?(@command)
      puts "Command required: populate_config_accounts, import_accounts, fund_accounts, stake_applications, or run"
      exit 1
    end
  end
end

# Run the script
options = Options.new
spammer = RelaySpammer.new(options)
spammer.run
